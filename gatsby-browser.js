/**
 * Implement Gatsby's Browser APIs in this file.
 *
 * See: https://www.gatsbyjs.org/docs/browser-apis/
 */

 // You can delete this file if you're not using it
 /* eslint-disable react/prop-types, import/no-extraneous-dependencies */
import React from 'react';
import { Router } from 'react-router-dom';
import OIDCProvider from './src/containers/OIDCProvider';

import oidc from './src/services/oidc';

exports.replaceRouterComponent = ({ history }) => {
  const ConnectedRouterWrapper = ({ children }) => (
    <OIDCProvider oidc={oidc}>
      <Router history={history}>{children}</Router>
    </OIDCProvider>
  );

  return ConnectedRouterWrapper;
};
