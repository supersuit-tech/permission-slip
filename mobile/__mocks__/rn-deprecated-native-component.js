'use strict';

// Blanket mock for react-native 0.85 deprecated component specs. The
// @react-native/babel-plugin-codegen 0.85 fails to parse several of these
// specs at jest transform time. These components are mocked by jest-expo
// at runtime anyway, so the spec files are never needed in tests.
//
// __esModule: true is required so Babel's interopRequireDefault treats
// `exports.default` as the ES module default (a string, valid as a host
// component type), rather than wrapping the whole exports object.
Object.defineProperty(exports, '__esModule', { value: true });
exports.Commands = {};
exports.default = 'RCTUnimplementedView';
