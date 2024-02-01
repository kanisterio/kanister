import { defineConfig } from "vitepress";

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Kanister",
  description: "Application-Specific Data Management",

  head: [["link", { rel: "icon", href: "favicon.ico" }]],

  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    logo: "kanister.svg",
    search: {
      provider: "local",
    },
    outline: [2, 3],
    // nav: [
    //   { text: "Home", link: "/", },
    //   { text: "Examples", link: "/markdown-examples" },
    //   { text: "API", link: "/api-examples", '' },
    // ],
    footer: {
      copyright: "© Copyright 2017-2024, Kanister",
    },

    sidebar: [
      //How we can group these
      { text: "Overview", link: "/overview" },
      { text: "Installation", link: "/install" },
      { text: "Tutorial", link: "/tutorial" },
      { text: "Architecture", link: "/architecture" },
      {
        text: "Tasks",
        collapsed: true,
        items: [
          {
            text: "Automating ActionSet Creation using Argo Cron Workflows",
            link: "/tasks/argo",
          },
          {
            text: "Segregate Controller And Datapath Logs",
            link: "/tasks/logs",
          },
          { text: "Modifying Kanister Log Level", link: "/tasks/logs_level" },
          {
            text: "Using ScaleWorkload function with output artifact",
            link: "/tasks/scaleworkload",
          },
        ],
      },
      { text: "Tooling", link: "/tooling" },
      { text: "Functions", link: "/functions" },
      { text: "Template Parameters", link: "/templates" },
      { text: "Troubleshooting", link: "/troubleshooting" },

      // { text: "Index 2", link: "/index 2" },
      // { text: "Index", link: "/index" },
      // { text: "Markdown-examples", link: "/markdown-examples" },
      // { text: "Api-examples", link: "/api-examples" },
    ],

    socialLinks: [
      { icon: "github", link: "https://github.com/kanisterio/kanister" },
    ],
  },
});
