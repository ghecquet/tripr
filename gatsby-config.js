module.exports = {
  siteMetadata: {
    title: 'Gatsby Default Starter',
    revealTheme: 'night',
    hljsTheme: 'monokai-sublime'
  },
  plugins: ['gatsby-plugin-react-helmet', 'gatsby-plugin-revealjs', {
      resolve: `gatsby-source-filesystem`,
      options: {
        name: `src`,
        path: `${__dirname}/src/`,
      },
    }]
}
