package store

import (
	"errors"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

type Store struct {
	DB *gorm.DB
}

// NewStore opens/creates the SQLite DB and runs migrations.
func NewStore(path string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate all models (add new ones here when needed)
	if err := db.AutoMigrate(
		&models.AppSettings{},
		&models.Domain{},
		&models.Sender{},
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

// UpsertSettings will create if no row exists, otherwise update.
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
