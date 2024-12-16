const fs = require("fs");
const path = require("path");

const ignoredFolders = ["_static", ".vitepress", "node_modules"];

function generateSidebarConfig(basePath, folderPath = "") {
  const folderDir = path.join(basePath, folderPath);
  const contents = fs.readdirSync(folderDir, { withFileTypes: true });

  const sidebarSections = contents
    .filter((item) => !ignoredFolders.includes(item.name))
    .map((item) => {
      const itemPath = path.join(folderPath, item.name);
      if (item.isDirectory()) {
        return {
          text: item.name.charAt(0).toUpperCase() + item.name.slice(1),
          items: generateSidebarConfig(basePath, itemPath),
        };
      } else if (item.isFile() && item.name.endsWith(".md")) {
        return {
          text:
            item.name.replace(/\.md$/, "").charAt(0).toUpperCase() +
            item.name.slice(1),
          link: `/${itemPath.replace(/\.md$/, "")}`,
        };
      }
      return null;
    });

  return sidebarSections.filter(Boolean);
}

const generatedSidebar = generateSidebarConfig("./");

fs.writeFileSync(
  "./generatedSidebar.js",
  `module.exports = ${JSON.stringify(generatedSidebar)};`
);
