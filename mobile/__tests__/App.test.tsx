import React from 'react';
import { act, create, ReactTestRenderer } from 'react-test-renderer';

jest.mock('../src/lib/supabaseClient', () => ({
  supabase: {
    auth: {
      getSession: jest.fn().mockResolvedValue({ data: { session: null }, error: null }),
      onAuthStateChange: jest.fn().mockReturnValue({ data: { subscription: { unsubscribe: jest.fn() } } }),
      signInWithOtp: jest.fn(),
      verifyOtp: jest.fn(),
      signOut: jest.fn(),
    },
  },
}));

jest.mock('expo-linking', () => ({
  createURL: jest.fn(() => 'permissionslip://'),
}));

jest.mock('expo-local-authentication', () => ({
  hasHardwareAsync: jest.fn().mockResolvedValue(false),
  isEnrolledAsync: jest.fn().mockResolvedValue(false),
  authenticateAsync: jest.fn().mockResolvedValue({ success: false }),
}));

jest.mock('expo-haptics', () => ({
  impactAsync: jest.fn(),
  notificationAsync: jest.fn(),
  ImpactFeedbackStyle: { Heavy: 'heavy' },
  NotificationFeedbackType: { Warning: 'warning' },
}));

import App from '../App';

describe('App', () => {
  let tree: ReactTestRenderer;

  afterEach(async () => {
    await act(async () => {
      tree?.unmount();
    });
  });

  it('renders without crashing', async () => {
    await act(async () => {
      tree = create(<App />);
    });
    expect(tree.toJSON()).toBeTruthy();
  });
});
