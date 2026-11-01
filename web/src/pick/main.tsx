import { createRoot } from "react-dom/client";
import { App } from "./App";
import "./pick.css";

const root = document.getElementById("root");
if (root) createRoot(root).render(<App />);
