# KumoMTA UI Testing Guide

## Overview

This project uses [Vitest](https://vitest.dev/) as the test runner and [@testing-library/react](https://testing-library.com/react) for React component testing.

## Running Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm test:coverage

# Run tests with UI
npm test:ui
```

## Test Structure

### Unit Tests
- Located alongside component files with `.test.jsx` or `.spec.jsx` extension
- Focus on testing individual component behavior and logic

### Test Setup
- `setup.js` - Global test configuration and mocks
- Includes mocks for browser APIs (clipboard, matchMedia, etc.)

## Writing Tests

### Example Test

```javascript
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MyComponent from './MyComponent';

describe('MyComponent', () => {
  it('should render correctly', async () => {
    render(<MyComponent />);
    
    await waitFor(() => {
      expect(screen.getByText('Expected Text')).toBeInTheDocument();
    });
  });
  
  it('should handle user interactions', async () => {
    const user = userEvent.setup();
    render(<MyComponent />);
    
    await user.click(screen.getByRole('button'));
    
    expect(screen.getByText('After Click')).toBeInTheDocument();
  });
});
```

### Best Practices

1. **Use `userEvent` over `fireEvent`** - More realistic user interactions
2. **Query by accessibility attributes** - Use `getByRole`, `getByLabelText`
3. **Wait for async updates** - Use `waitFor` for async operations
4. **Mock external dependencies** - Use `vi.mock()` for API calls
5. **Test user behavior, not implementation** - Focus on what users see and do

## Mocking

### API Mocks

```javascript
vi.mock('../api', () => ({
  fetchData: vi.fn(),
}));
```

### Component Mocks

```javascript
vi.mock('lucide-react', () => ({
  Icon: () => <span data-testid="icon" />,
}));
```

## Coverage Goals

- Aim for >80% coverage on critical paths
- 100% coverage on utility functions
- Focus on edge cases and error handling

## Common Patterns

### Testing Forms
- Test validation
- Test submission
- Test error handling
- Test reset/cancel behavior

### Testing Lists
- Test empty state
- Test item rendering
- Test item actions (edit, delete)
- Test filtering/searching

### Testing Modals
- Test open/close
- Test form submission
- Test outside click behavior

## Troubleshooting

### Tests timing out
- Increase timeout: `{ timeout: 10000 }`
- Check for missing `waitFor` calls
- Verify async operations complete

### Elements not found
- Use `screen.debug()` to see rendered HTML
- Check query methods (getBy vs queryBy vs findBy)
- Ensure component has time to render

### Mock not working
- Verify mock path is correct
- Check mock is defined before import
- Use `vi.clearAllMocks()` in `beforeEach`

## Resources

- [Vitest Documentation](https://vitest.dev/)
- [Testing Library Docs](https://testing-library.com/)
- [Testing Library Cheatsheet](https://testing-library.com/docs/react-testing-library/cheatsheet)