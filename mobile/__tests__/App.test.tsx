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
