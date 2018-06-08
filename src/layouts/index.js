import React from 'react'
import PropTypes from 'prop-types'
import Helmet from 'react-helmet'

import Header from '../components/header'
import './index.css'

const Layout = ({ children, data }) => (
  <div>
    <Helmet
      title={data.site.siteMetadata.title}
      meta={[
        { name: 'description', content: 'Sample' },
        { name: 'keywords', content: 'sample, something' },
      ]}
    >
      <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/reveal.js/3.6.0/css/reveal.min.css" integrity="sha256-ol2N5Xr80jdDqxK0lte3orKGb9Ja3sOnpAUV7TTADmg=" crossorigin="anonymous" />

      <link rel="stylesheet" href={"https://cdnjs.cloudflare.com/ajax/libs/reveal.js/3.6.0/css/theme/" + data.site.siteMetadata.revealTheme + ".min.css"} />
      <link rel="stylesheet" href={"https://cdnjs.cloudflare.com/ajax/libs/highlight.js/9.12.0/styles/" + data.site.siteMetadata.hljsTheme + ".min.css"} />
    </Helmet>
    <Header siteTitle={data.site.siteMetadata.title} />
    <div
      style={{
        margin: '0 auto',
        maxWidth: 960,
        padding: '0px 1.0875rem 1.45rem',
        paddingTop: 0,
      }}
    >
      {children()}
    </div>
  </div>
)

Layout.propTypes = {
  children: PropTypes.func,
}

export default Layout

export const query = graphql`
  query SiteTitleQuery {
    site {
      siteMetadata {
        title
      }
    }
  }
`
