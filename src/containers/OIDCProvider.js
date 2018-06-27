// ./src/containers/OIDCProvider.js
import React from 'react';
import PropTypes from 'prop-types';

class OIDCProvider extends React.Component {
  static propTypes = {
    children: PropTypes.element,
    oidc: PropTypes.object.isRequired,
  };

  static childContextTypes = {
    oidc: PropTypes.object,
  };

  getChildContext() {
    const { oidc } = this.props;

    return {
      oidc,
    };
  }

  render() {
    const { children } = this.props;

    return children;
  }
}

export default OIDCProvider;
