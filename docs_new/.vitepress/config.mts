import { defineConfig } from "vitepress";
import version from "./version.json"

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
    footer: {
      copyright: "© Copyright 2017-2024, Kanister",
    },
    nav: [{text: version.version, link: ""}],

    sidebar: [
      { text: "Overview", link: "/overview" },
      { text: "Installation", link: "/install" },
      { text: "Tutorial", link: "/tutorial" },
      { text: "Architecture", link: "/architecture" },
      { text: "Tooling", link: "/tooling" },
      { text: "Functions", link: "/functions" },
      { text: "Template Parameters", link: "/templates" },
      { text: "Troubleshooting", link: "/troubleshooting" },
      {
        text: "Tasks",
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
          { text: "Configuring logs for specific ActionSets", link: "/tasks/logs_labels" },
          {
            text: "Using ScaleWorkload function with output artifact",
            link: "/tasks/scaleworkload",
          },
        ],
      },
    ],

    socialLinks: [
      { icon: "github", link: "https://github.com/kanisterio/kanister" },
    ],
  },
});
