// To see all options:
// https://vuepress.vuejs.org/config/
// https://vuepress.vuejs.org/theme/default-theme-config.html
module.exports = {
  title: "goldilocks Documentation",
  description: "Documentation for Fairwinds' goldilocks",
  themeConfig: {
    docsRepo: "FairwindsOps/goldilocks",
    sidebar: [
      {
        title: "Pluto",
        path: "/",
        sidebarDepth: 0,
      },
      {
        title: "Installation",
        path: "/installation",
      },
      {
        title: "Advanced Usage",
        path: "/advanced",
      },
      {
        title: "Contributing",
        children: [
          {
            title: "Guide",
            path: "contributing/guide"
          },
          {
            title: "Code of Conduct",
            path: "contributing/code-of-conduct"
          }
        ]
      }
    ]
  },
}
