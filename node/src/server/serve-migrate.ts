import express from "express";
import path from "path";

export const startServer = () => {
  const app = express();
  const PORT = process.env.PORT || 3000;

  // Go one level up from __dirname to reach project root
  const publicPath = path.resolve(__dirname, "../public");

  // Serve static files
  app.use(express.static(publicPath));

  // Optional: Explicit route to migrate.html
  app.get("/*", (req, res) => {
    res.sendFile(path.join(publicPath, "migrate.html"));
  });

  app.listen(PORT, () => {
    console.log(`Server running at http://localhost:${PORT}`);
  });
};
