import {UserManager} from "oidc-client"

var config = {
    authority: 'http://127.0.0.1:53205/dex',
    client_id: "cells-front",
    redirect_uri: "http://127.0.0.1:5555/callback",
    response_type: "code",
    scope: "openid"
};

class OpenIDConnect {
  constructor() {
      this.userManager = new UserManager(config);
  }
}

export default new OpenIDConnect();
