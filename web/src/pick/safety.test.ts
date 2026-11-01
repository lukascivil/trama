import { readFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

// What GitHub sends (names, descriptions) is text from other people. The page
// must draw it as text. The bundle carries React's own code, which names
// innerHTML, so the check reads the page's source instead.
describe("the picker page source", () => {
  const dir = dirname(fileURLToPath(import.meta.url));
  const sources = readdirSync(dir).filter((f) => /\.tsx?$/.test(f) && !f.endsWith(".test.ts"));

  it("has sources to check", () => {
    expect(sources).toContain("App.tsx");
  });

  for (const file of sources) {
    it(`${file} never injects HTML`, () => {
      const text = readFileSync(join(dir, file), "utf8");
      for (const sink of ["dangerouslySetInnerHTML", "innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval("]) {
        expect(text, `${file} uses ${sink}`).not.toContain(sink);
      }
    });
  }
});
