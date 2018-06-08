import Reveal from "reveal.js"

exports.onRenderBody = (
  { setHeadComponents, setHtmlAttributes, setBodyAttributes },
  pluginOptions
) => {
  Reveal.initialize()
}
