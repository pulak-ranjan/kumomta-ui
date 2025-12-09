package store

import (
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

type Store struct {
	DB *gorm.DB
}

func NewStore(path string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.AppSettings{},
		&models.Domain{},
		&models.Sender{},
		&models.AdminUser{},
		&models.AuthSession{}, // <--- Added
		&models.BounceAccount{},
		&models.SystemIP{},
	); err != nil {
		return nil, err
	}

	return &Store{DB: db}, nil
}

func (s *Store) LogError(err error) {
	if err != nil {
		log.Println("[STORE ERROR]", err)
	}
}

var ErrNotFound = gorm.ErrRecordNotFound

// ----------------------
// Auth Sessions (Multi-Device)
// ----------------------

// CreateSession creates a new token and ensures max 3 active sessions
func (s *Store) CreateSession(adminID uint, token string, ip string, duration time.Duration) error {
	// 1. Cleanup expired
	s.DB.Where("expires_at < ?", time.Now()).Delete(&models.AuthSession{})

	// 2. Enforce Max 3 Limit (Delete oldest if needed)
	var count int64
	s.DB.Model(&models.AuthSession{}).Where("admin_id = ?", adminID).Count(&count)
	if count >= 3 {
		var oldest models.AuthSession
		s.DB.Where("admin_id = ?", adminID).Order("created_at asc").First(&oldest)
		if oldest.ID != 0 {
			s.DB.Delete(&oldest)
		}
	}

	// 3. Create New
	sess := models.AuthSession{
		AdminID:   adminID,
		Token:     token,
		ExpiresAt: time.Now().Add(duration),
		DeviceIP:  ip,
	}
	return s.DB.Create(&sess).Error
}

func (s *Store) GetAdminBySessionToken(token string) (*models.AdminUser, error) {
	var sess models.AuthSession
	err := s.DB.Where("token = ? AND expires_at > ?", token, time.Now()).First(&sess).Error
	if err != nil {
		return nil, err
	}

	var admin models.AdminUser
	err = s.DB.First(&admin, sess.AdminID).Error
	return &admin, err
}

func (s *Store) DeleteSession(token string) error {
	return s.DB.Where("token = ?", token).Delete(&models.AuthSession{}).Error
}

// ----------------------
// App Settings
// ----------------------

func (s *Store) GetSettings() (*models.AppSettings, error) {
	var st models.AppSettings
	err := s.DB.First(&st).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Store) UpsertSettings(st *models.AppSettings) error {
	if st.ID == 0 {
		return s.DB.Create(st).Error
	}
	return s.DB.Save(st).Error
}

// ----------------------
// Domains
// ----------------------

func (s *Store) ListDomains() ([]models.Domain, error) {
	var domains []models.Domain
	err := s.DB.Preload("Senders").Find(&domains).Error
	return domains, err
}

func (s *Store) GetDomainByID(id uint) (*models.Domain, error) {
	var d models.Domain
	err := s.DB.Preload("Senders").First(&d, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) GetDomainByName(name string) (*models.Domain, error) {
	var d models.Domain
	err := s.DB.Preload("Senders").Where("name = ?", name).First(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) CreateDomain(d *models.Domain) error {
	return s.DB.Create(d).Error
}

func (s *Store) UpdateDomain(d *models.Domain) error {
	return s.DB.Save(d).Error
}

func (s *Store) DeleteDomain(id uint) error {
	return s.DB.Delete(&models.Domain{}, id).Error
}

func (s *Store) CountDomains() (int64, error) {
	var c int64
	err := s.DB.Model(&models.Domain{}).Count(&c).Error
	return c, err
}

// ----------------------
// Senders
// ----------------------

func (s *Store) ListSendersByDomain(domainID uint) ([]models.Sender, error) {
	var senders []models.Sender
	err := s.DB.Where("domain_id = ?", domainID).Find(&senders).Error
	return senders, err
}

func (s *Store) GetSenderByID(id uint) (*models.Sender, error) {
	var snd models.Sender
	err := s.DB.First(&snd, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &snd, nil
}

func (s *Store) CreateSender(snd *models.Sender) error {
	return s.DB.Create(snd).Error
}

func (s *Store) UpdateSender(snd *models.Sender) error {
	return s.DB.Save(snd).Error
}

func (s *Store) DeleteSender(id uint) error {
	return s.DB.Delete(&models.Sender{}, id).Error
}

func (s *Store) CountSenders() (int64, error) {
	var c int64
	err := s.DB.Model(&models.Sender{}).Count(&c).Error
	return c, err
}

// ----------------------
// Admin Users
// ----------------------

func (s *Store) AdminCount() (int64, error) {
	var count int64
	err := s.DB.Model(&models.AdminUser{}).Count(&count).Error
	return count, err
}

func (s *Store) GetAdminByEmail(email string) (*models.AdminUser, error) {
	var u models.AdminUser
	err := s.DB.Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) CreateAdmin(u *models.AdminUser) error {
	return s.DB.Create(u).Error
}

func (s *Store) UpdateAdmin(u *models.AdminUser) error {
	return s.DB.Save(u).Error
}

// ----------------------
// Bounce Accounts
// ----------------------

func (s *Store) ListBounceAccounts() ([]models.BounceAccount, error) {
	var list []models.BounceAccount
	err := s.DB.Find(&list).Error
	return list, err
}

func (s *Store) GetBounceAccountByID(id uint) (*models.BounceAccount, error) {
	var b models.BounceAccount
	err := s.DB.First(&b, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) CreateBounceAccount(b *models.BounceAccount) error {
	return s.DB.Create(b).Error
}

func (s *Store) UpdateBounceAccount(b *models.BounceAccount) error {
	return s.DB.Save(b).Error
}

func (s *Store) DeleteBounceAccount(id uint) error {
	return s.DB.Delete(&models.BounceAccount{}, id).Error
}

// ----------------------
// System IPs
// ----------------------

func (s *Store) ListSystemIPs() ([]models.SystemIP, error) {
	var list []models.SystemIP
	err := s.DB.Find(&list).Error
	return list, err
}

func (s *Store) CreateSystemIP(ip *models.SystemIP) error {
	return s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(ip).Error
}

func (s *Store) CreateSystemIPs(ips []models.SystemIP) error {
	if len(ips) == 0 {
		return nil
	}
	return s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&ips).Error
}

func (s *Store) DeleteSystemIP(id uint) error {
	return s.DB.Delete(&models.SystemIP{}, id).Error
}
