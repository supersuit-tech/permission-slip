import React from 'react';
import { act, create, ReactTestRenderer } from 'react-test-renderer';
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
