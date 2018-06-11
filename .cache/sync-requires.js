// prefer default export if available
const preferDefault = m => m && m.default || m


exports.layouts = {
  "layout---index": preferDefault(require("/Users/gregory/Sites/tripr/.cache/layouts/index.js"))
}

exports.components = {
  "component---cache-dev-404-page-js": preferDefault(require("/Users/gregory/Sites/tripr/.cache/dev-404-page.js")),
  "component---src-pages-404-js": preferDefault(require("/Users/gregory/Sites/tripr/src/pages/404.js")),
  "component---src-pages-index-js": preferDefault(require("/Users/gregory/Sites/tripr/src/pages/index.js")),
  "component---src-pages-page-2-js": preferDefault(require("/Users/gregory/Sites/tripr/src/pages/page-2.js")),
  "component---src-pages-slides-slide-1-jsx": preferDefault(require("/Users/gregory/Sites/tripr/src/pages/slides/slide1.jsx")),
  "component---src-pages-slides-slide-2-jsx": preferDefault(require("/Users/gregory/Sites/tripr/src/pages/slides/slide2.jsx"))
}

exports.json = {
  "layout-index.json": require("/Users/gregory/Sites/tripr/.cache/json/layout-index.json"),
  "dev-404-page.json": require("/Users/gregory/Sites/tripr/.cache/json/dev-404-page.json"),
  "404.json": require("/Users/gregory/Sites/tripr/.cache/json/404.json"),
  "index.json": require("/Users/gregory/Sites/tripr/.cache/json/index.json"),
  "page-2.json": require("/Users/gregory/Sites/tripr/.cache/json/page-2.json"),
  "slides-slide-1.json": require("/Users/gregory/Sites/tripr/.cache/json/slides-slide-1.json"),
  "404-html.json": require("/Users/gregory/Sites/tripr/.cache/json/404-html.json"),
  "slides-slide-2.json": require("/Users/gregory/Sites/tripr/.cache/json/slides-slide-2.json")
}