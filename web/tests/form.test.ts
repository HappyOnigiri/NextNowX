import { describe, expect, it } from "vitest";
import { formValue } from "../src/form";

describe("formValue", () => {
  it("returns text values and rejects missing or file values", () => {
    const data = new FormData();
    data.set("title", "Release Next Now X");
    data.set("attachment", new File(["content"], "notes.txt"));

    expect(formValue(data, "title")).toBe("Release Next Now X");
    expect(formValue(data, "attachment")).toBe("");
    expect(formValue(data, "missing")).toBe("");
  });
});
