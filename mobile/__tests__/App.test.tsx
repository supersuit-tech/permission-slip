import React from 'react';
import { act, create } from 'react-test-renderer';
import App from '../App';

describe('App', () => {
  it('renders without crashing', async () => {
    let tree;
    await act(async () => {
      tree = create(<App />);
    });
    expect(tree!.toJSON()).toBeTruthy();
  });
});
