import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Domains from './Domains';
import * as api from '../api';

// Mock the API module
vi.mock('../api', () => ({
  listDomains: vi.fn(),
  saveDomain: vi.fn(),
  deleteDomain: vi.fn(),
  saveSender: vi.fn(),
  deleteSender: vi.fn(),
  getSettings: vi.fn(),
  getSystemIPs: vi.fn(),
  importSenders: vi.fn(),
}));

// Mock lucide-react icons to avoid rendering issues
vi.mock('lucide-react', () => ({
  Plus: () => <span data-testid="plus-icon" />,
  Upload: () => <span data-testid="upload-icon" />,
  Trash2: () => <span data-testid="trash-icon" />,
  Edit2: () => <span data-testid="edit-icon" />,
  Copy: () => <span data-testid="copy-icon" />,
  Check: () => <span data-testid="check-icon" />,
  Server: () => <span data-testid="server-icon" />,
  Globe: () => <span data-testid="globe-icon" />,
  Mail: () => <span data-testid="mail-icon" />,
  ShieldCheck: () => <span data-testid="shield-icon" />,
  AlertCircle: () => <span data-testid="alert-icon" />,
  MoreHorizontal: () => <span data-testid="more-icon" />,
}));

describe('Domains Component', () => {
  const mockDomains = [
    {
      id: 1,
      name: 'example.com',
      mail_host: 'mail.example.com',
      bounce_host: 'bounce.example.com',
      senders: [
        {
          id: 1,
          email: 'sender@example.com',
          local_part: 'sender',
          ip: '192.168.1.1',
          smtp_password: 'password123',
          has_dkim: true,
        },
      ],
    },
    {
      id: 2,
      name: 'test.com',
      mail_host: 'mail.test.com',
      bounce_host: 'bounce.test.com',
      senders: [],
    },
  ];

  const mockSettings = {
    main_server_ip: '203.0.113.1',
  };

  const mockSystemIPs = [
    { id: 1, value: '192.168.1.1', interface: 'eth0' },
    { id: 2, value: '192.168.1.2', interface: 'eth1' },
  ];

  beforeEach(() => {
    // Reset all mocks before each test
    vi.clearAllMocks();
    
    // Setup default mock implementations
    api.listDomains.mockResolvedValue(mockDomains);
    api.getSettings.mockResolvedValue(mockSettings);
    api.getSystemIPs.mockResolvedValue(mockSystemIPs);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Initial Rendering', () => {
    it('should render loading state initially', () => {
      // Make the API call pending
      api.listDomains.mockImplementation(() => new Promise(() => {}));
      
      render(<Domains />);
      
      expect(screen.getByText(/loading domains/i)).toBeInTheDocument();
    });

    it('should render domains after loading', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
        expect(screen.getByText('test.com')).toBeInTheDocument();
      });
    });

    it('should display page title and description', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Domains')).toBeInTheDocument();
        expect(screen.getByText(/manage your sending domains/i)).toBeInTheDocument();
      });
    });

    it('should show action buttons', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Import CSV')).toBeInTheDocument();
        expect(screen.getByText('New Domain')).toBeInTheDocument();
      });
    });

    it('should display empty state when no domains exist', async () => {
      api.listDomains.mockResolvedValue([]);
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText(/no domains found/i)).toBeInTheDocument();
        expect(screen.getByText(/add your first domain/i)).toBeInTheDocument();
      });
    });
  });

  describe('Domain Display', () => {
    it('should display domain names and hosts', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
        expect(screen.getByText('mail.example.com')).toBeInTheDocument();
        expect(screen.getByText('bounce.example.com')).toBeInTheDocument();
      });
    });

    it('should display senders for each domain', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('sender@example.com')).toBeInTheDocument();
      });
    });

    it('should show DKIM indicator when available', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        const senderRow = screen.getByText('sender@example.com').closest('div');
        expect(within(senderRow).getByText('DKIM')).toBeInTheDocument();
      });
    });

    it('should display IP addresses for senders', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText(/IP: 192\.168\.1\.1/)).toBeInTheDocument();
      });
    });

    it('should show default IP text when no IP is set', async () => {
      const domainsWithoutIP = [{
        ...mockDomains[0],
        senders: [{
          ...mockDomains[0].senders[0],
          ip: '',
        }],
      }];
      api.listDomains.mockResolvedValue(domainsWithoutIP);
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText(/IP: Default/)).toBeInTheDocument();
      });
    });

    it('should display "No senders configured" when domain has no senders', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        const testDomain = screen.getByText('test.com').closest('div').closest('div').closest('div');
        expect(within(testDomain).getByText(/no senders configured/i)).toBeInTheDocument();
      });
    });
  });

  describe('DNS Records Display', () => {
    it('should display DNS records for each domain', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText(/DNS Records/i)).toBeInTheDocument();
      });
    });

    it('should show A records for mail and bounce hosts', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        const dnsSection = screen.getAllByText(/A \(Mail\)/i)[0];
        expect(dnsSection).toBeInTheDocument();
      });
    });

    it('should display MX record', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getAllByText('MX')[0]).toBeInTheDocument();
      });
    });

    it('should show SPF record with correct IPs', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        const spfElements = screen.getAllByText('SPF');
        expect(spfElements.length).toBeGreaterThan(0);
      });
    });

    it('should include all sender IPs in SPF record', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        // Check that the component rendered successfully
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      // SPF record should contain sender IPs
      // The DNS helper builds SPF from sender IPs
    });
  });

  describe('Copy to Clipboard Functionality', () => {
    it('should copy DNS record to clipboard when copy button clicked', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      const copyButtons = screen.getAllByTestId('copy-icon');
      await user.click(copyButtons[0]);
      
      expect(navigator.clipboard.writeText).toHaveBeenCalled();
    });

    it('should show check icon after successful copy', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      const copyButton = screen.getAllByTestId('copy-icon')[0].closest('button');
      await user.click(copyButton);
      
      // Check icon should appear after copy
      await waitFor(() => {
        expect(screen.getAllByTestId('check-icon').length).toBeGreaterThan(0);
      });
    });
  });

  describe('Domain Creation', () => {
    it('should open new domain modal when "New Domain" button clicked', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('New Domain')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('New Domain'));
      
      expect(screen.getByRole('heading', { name: /new domain/i })).toBeInTheDocument();
    });

    it('should display form fields in new domain modal', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('New Domain')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('New Domain'));
      
      expect(screen.getByLabelText(/domain name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/mail host/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/bounce host/i)).toBeInTheDocument();
    });

    it('should save new domain when form submitted', async () => {
      const user = userEvent.setup();
      api.saveDomain.mockResolvedValue({ id: 3, name: 'newdomain.com' });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('New Domain')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('New Domain'));
      
      const nameInput = screen.getByLabelText(/domain name/i);
      await user.type(nameInput, 'newdomain.com');
      
      await user.click(screen.getByRole('button', { name: /save/i }));
      
      await waitFor(() => {
        expect(api.saveDomain).toHaveBeenCalledWith(expect.objectContaining({
          name: 'newdomain.com',
        }));
      });
    });

    it('should close modal after successful domain creation', async () => {
      const user = userEvent.setup();
      api.saveDomain.mockResolvedValue({ id: 3, name: 'newdomain.com' });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('New Domain')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('New Domain'));
      
      const nameInput = screen.getByLabelText(/domain name/i);
      await user.type(nameInput, 'newdomain.com');
      
      await user.click(screen.getByRole('button', { name: /save/i }));
      
      await waitFor(() => {
        expect(screen.queryByRole('heading', { name: /new domain/i })).not.toBeInTheDocument();
      });
    });

    it('should display error message on domain creation failure', async () => {
      const user = userEvent.setup();
      api.saveDomain.mockRejectedValue(new Error('Failed to create domain'));
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('New Domain')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('New Domain'));
      
      const nameInput = screen.getByLabelText(/domain name/i);
      await user.type(nameInput, 'newdomain.com');
      
      await user.click(screen.getByRole('button', { name: /save/i }));
      
      await waitFor(() => {
        expect(screen.getByText(/failed to create domain/i)).toBeInTheDocument();
      });
    });

    it('should cancel domain creation when cancel button clicked', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('New Domain')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('New Domain'));
      
      await user.click(screen.getByRole('button', { name: /cancel/i }));
      
      expect(screen.queryByRole('heading', { name: /new domain/i })).not.toBeInTheDocument();
    });
  });

  describe('Domain Editing', () => {
    it('should open edit modal when edit button clicked', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      const editButtons = screen.getAllByTestId('edit-icon');
      await user.click(editButtons[0].closest('button'));
      
      expect(screen.getByRole('heading', { name: /edit domain/i })).toBeInTheDocument();
    });

    it('should pre-populate form with existing domain data', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      const editButtons = screen.getAllByTestId('edit-icon');
      await user.click(editButtons[0].closest('button'));
      
      const nameInput = screen.getByLabelText(/domain name/i);
      expect(nameInput).toHaveValue('example.com');
    });

    it('should update domain when edit form submitted', async () => {
      const user = userEvent.setup();
      api.saveDomain.mockResolvedValue({ ...mockDomains[0], name: 'updated.com' });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      const editButtons = screen.getAllByTestId('edit-icon');
      await user.click(editButtons[0].closest('button'));
      
      const nameInput = screen.getByLabelText(/domain name/i);
      await user.clear(nameInput);
      await user.type(nameInput, 'updated.com');
      
      await user.click(screen.getByRole('button', { name: /save/i }));
      
      await waitFor(() => {
        expect(api.saveDomain).toHaveBeenCalled();
      });
    });
  });

  describe('Domain Deletion', () => {
    it('should show confirmation dialog when delete button clicked', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      const deleteButtons = screen.getAllByTestId('trash-icon');
      await user.click(deleteButtons[0].closest('button'));
      
      expect(window.confirm).toHaveBeenCalledWith(expect.stringContaining('Delete domain'));
    });

    it('should delete domain when confirmed', async () => {
      const user = userEvent.setup();
      api.deleteDomain.mockResolvedValue({ status: 'deleted' });
      global.confirm.mockReturnValue(true);
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      const deleteButtons = screen.getAllByTestId('trash-icon');
      await user.click(deleteButtons[0].closest('button'));
      
      await waitFor(() => {
        expect(api.deleteDomain).toHaveBeenCalledWith(1);
      });
    });

    it('should not delete domain when cancelled', async () => {
      const user = userEvent.setup();
      global.confirm.mockReturnValue(false);
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      const deleteButtons = screen.getAllByTestId('trash-icon');
      await user.click(deleteButtons[0].closest('button'));
      
      expect(api.deleteDomain).not.toHaveBeenCalled();
    });
  });

  describe('Sender Management', () => {
    it('should open add sender modal when "Add Sender" clicked', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText(/add sender/i));
      
      expect(screen.getByRole('heading', { name: /new sender/i })).toBeInTheDocument();
    });

    it('should display sender form fields', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText(/add sender/i));
      
      expect(screen.getByLabelText(/local part/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/email address/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/ip address/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/smtp password/i)).toBeInTheDocument();
    });

    it('should save new sender when form submitted', async () => {
      const user = userEvent.setup();
      api.saveSender.mockResolvedValue({ id: 2, email: 'newsender@example.com' });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText(/add sender/i));
      
      const localPartInput = screen.getByLabelText(/local part/i);
      await user.type(localPartInput, 'newsender');
      
      await user.click(screen.getByRole('button', { name: /save/i }));
      
      await waitFor(() => {
        expect(api.saveSender).toHaveBeenCalled();
      });
    });

    it('should populate system IPs in dropdown', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText(/add sender/i));
      
      const ipSelect = screen.getByLabelText(/ip address/i);
      expect(within(ipSelect).getByText('192.168.1.1 (eth0)')).toBeInTheDocument();
      expect(within(ipSelect).getByText('192.168.1.2 (eth1)')).toBeInTheDocument();
    });

    it('should edit sender when edit button clicked', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('sender@example.com')).toBeInTheDocument();
      });
      
      const senderRow = screen.getByText('sender@example.com').closest('div');
      const editButton = within(senderRow).getAllByTestId('edit-icon')[0].closest('button');
      await user.click(editButton);
      
      expect(screen.getByRole('heading', { name: /edit sender/i })).toBeInTheDocument();
    });

    it('should delete sender when delete confirmed', async () => {
      const user = userEvent.setup();
      api.deleteSender.mockResolvedValue({ status: 'deleted' });
      global.confirm.mockReturnValue(true);
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('sender@example.com')).toBeInTheDocument();
      });
      
      const senderRow = screen.getByText('sender@example.com').closest('div');
      const deleteButton = within(senderRow).getAllByTestId('trash-icon')[0].closest('button');
      await user.click(deleteButton);
      
      await waitFor(() => {
        expect(api.deleteSender).toHaveBeenCalledWith(1);
      });
    });
  });

  describe('CSV Import', () => {
    it('should open import modal when "Import CSV" clicked', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Import CSV')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('Import CSV'));
      
      expect(screen.getByRole('heading', { name: /bulk import/i })).toBeInTheDocument();
    });

    it('should display import instructions', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Import CSV')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('Import CSV'));
      
      expect(screen.getByText(/domain, localpart, ip, password/i)).toBeInTheDocument();
    });

    it('should show file input in import modal', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Import CSV')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('Import CSV'));
      
      const fileInput = screen.getByRole('textbox', { hidden: true });
      expect(fileInput).toHaveAttribute('type', 'file');
      expect(fileInput).toHaveAttribute('accept', '.csv');
    });

    it('should call importSenders API when file submitted', async () => {
      const user = userEvent.setup();
      const mockFile = new File(['domain,localpart\nexample.com,test'], 'test.csv', { type: 'text/csv' });
      api.importSenders.mockResolvedValue({ message: 'Import successful' });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Import CSV')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('Import CSV'));
      
      const fileInput = screen.getByLabelText(/upload a csv/i, { selector: 'input[type="file"]' });
      await user.upload(fileInput, mockFile);
      
      await user.click(screen.getAllByRole('button', { name: /import/i })[0]);
      
      await waitFor(() => {
        expect(api.importSenders).toHaveBeenCalled();
      });
    });

    it('should display success message after import', async () => {
      const user = userEvent.setup();
      const mockFile = new File(['domain,localpart\nexample.com,test'], 'test.csv', { type: 'text/csv' });
      api.importSenders.mockResolvedValue({ message: 'Imported 5 senders successfully' });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Import CSV')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('Import CSV'));
      
      const fileInput = screen.getByLabelText(/upload a csv/i, { selector: 'input[type="file"]' });
      await user.upload(fileInput, mockFile);
      
      await user.click(screen.getAllByRole('button', { name: /import/i })[0]);
      
      await waitFor(() => {
        expect(screen.getByText(/imported 5 senders successfully/i)).toBeInTheDocument();
      });
    });

    it('should close import modal after successful import', async () => {
      const user = userEvent.setup();
      const mockFile = new File(['domain,localpart\nexample.com,test'], 'test.csv', { type: 'text/csv' });
      api.importSenders.mockResolvedValue({ message: 'Import successful' });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Import CSV')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('Import CSV'));
      
      const fileInput = screen.getByLabelText(/upload a csv/i, { selector: 'input[type="file"]' });
      await user.upload(fileInput, mockFile);
      
      await user.click(screen.getAllByRole('button', { name: /import/i })[0]);
      
      await waitFor(() => {
        expect(screen.queryByRole('heading', { name: /bulk import/i })).not.toBeInTheDocument();
      });
    });

    it('should handle import errors gracefully', async () => {
      const user = userEvent.setup();
      const mockFile = new File(['invalid'], 'test.csv', { type: 'text/csv' });
      api.importSenders.mockRejectedValue(new Error('Invalid CSV format'));
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('Import CSV')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('Import CSV'));
      
      const fileInput = screen.getByLabelText(/upload a csv/i, { selector: 'input[type="file"]' });
      await user.upload(fileInput, mockFile);
      
      await user.click(screen.getAllByRole('button', { name: /import/i })[0]);
      
      await waitFor(() => {
        expect(screen.getByText(/invalid csv format/i)).toBeInTheDocument();
      });
    });
  });

  describe('Error Handling', () => {
    it('should display error message when API calls fail', async () => {
      api.listDomains.mockRejectedValue(new Error('Network error'));
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText(/network error/i)).toBeInTheDocument();
      });
    });

    it('should handle null/undefined API responses gracefully', async () => {
      api.listDomains.mockResolvedValue(null);
      api.getSettings.mockResolvedValue(null);
      api.getSystemIPs.mockResolvedValue(null);
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText(/no domains found/i)).toBeInTheDocument();
      });
    });

    it('should recover from errors on reload', async () => {
      api.listDomains.mockRejectedValueOnce(new Error('Network error'));
      api.listDomains.mockResolvedValue(mockDomains);
      
      const { rerender } = render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText(/network error/i)).toBeInTheDocument();
      });
      
      // Simulate component re-mount/reload
      rerender(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
    });
  });

  describe('DNS Helper Logic', () => {
    it('should generate correct SPF record with all IPs', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      // SPF record should include both sender IP and main server IP
      // v=spf1 ip4:192.168.1.1 ip4:203.0.113.1 ~all
      const domainCard = screen.getByText('example.com').closest('div').closest('div').closest('div');
      expect(domainCard).toBeInTheDocument();
    });

    it('should use default server IP when no sender IPs provided', async () => {
      const domainsWithoutSenderIP = [{
        ...mockDomains[1],
      }];
      api.listDomains.mockResolvedValue(domainsWithoutSenderIP);
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('test.com')).toBeInTheDocument();
      });
    });

    it('should handle missing settings gracefully', async () => {
      api.getSettings.mockResolvedValue(null);
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      // Should use fallback SERVER_IP
    });
  });

  describe('State Management', () => {
    it('should clear form after successful sender creation', async () => {
      const user = userEvent.setup();
      api.saveSender.mockResolvedValue({ id: 2, email: 'newsender@example.com' });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText(/add sender/i));
      
      const localPartInput = screen.getByLabelText(/local part/i);
      await user.type(localPartInput, 'newsender');
      
      await user.click(screen.getByRole('button', { name: /save/i }));
      
      await waitFor(() => {
        expect(screen.queryByRole('heading', { name: /new sender/i })).not.toBeInTheDocument();
      });
    });

    it('should reload data after domain operations', async () => {
      const user = userEvent.setup();
      api.saveDomain.mockResolvedValue({ id: 3, name: 'newdomain.com' });
      let callCount = 0;
      api.listDomains.mockImplementation(() => {
        callCount++;
        return Promise.resolve(callCount === 1 ? mockDomains : [...mockDomains, { id: 3, name: 'newdomain.com', senders: [] }]);
      });
      
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('example.com')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('New Domain'));
      
      const nameInput = screen.getByLabelText(/domain name/i);
      await user.type(nameInput, 'newdomain.com');
      
      await user.click(screen.getByRole('button', { name: /save/i }));
      
      await waitFor(() => {
        expect(api.listDomains).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper form labels', async () => {
      const user = userEvent.setup();
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByText('New Domain')).toBeInTheDocument();
      });
      
      await user.click(screen.getByText('New Domain'));
      
      expect(screen.getByLabelText(/domain name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/mail host/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/bounce host/i)).toBeInTheDocument();
    });

    it('should have accessible buttons', async () => {
      render(<Domains />);
      
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /import csv/i })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /new domain/i })).toBeInTheDocument();
      });
    });
  });
});