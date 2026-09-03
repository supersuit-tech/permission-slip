import { describe, it, expect } from "vitest";
import {
  isBase64ParamKey,
  formatBinaryParamSummary,
  binaryThumbnailSrc,
  decodedBase64Size,
  formatByteSize,
} from "../binaryParamDisplay";

const PNG_1PX =
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==";

describe("isBase64ParamKey", () => {
  it("matches content_base64 and other *_base64 keys", () => {
    expect(isBase64ParamKey("content_base64")).toBe(true);
    expect(isBase64ParamKey("file_base64")).toBe(true);
    expect(isBase64ParamKey("content")).toBe(false);
    expect(isBase64ParamKey("mime_type")).toBe(false);
  });
});

describe("formatBinaryParamSummary", () => {
  it("summarizes a PDF using mime_type and decoded size", () => {
    const pdf = "JVBERi0xLjQK";
    expect(
      formatBinaryParamSummary(pdf, { mime_type: "application/pdf" }),
    ).toBe(`PDF \u00b7 ${formatByteSize(decodedBase64Size(pdf))}`);
  });

  it("detects PDF from magic bytes when mime_type is absent", () => {
    const pdf = "JVBERi0xLjQK";
    expect(formatBinaryParamSummary(pdf, {})).toMatch(/^PDF /);
  });

  it("labels a PNG from magic bytes", () => {
    expect(formatBinaryParamSummary(PNG_1PX, {})).toMatch(/^PNG image /);
  });
});

describe("binaryThumbnailSrc", () => {
  it("returns a data URI for small images", () => {
    const src = binaryThumbnailSrc("content_base64", PNG_1PX, {});
    expect(src).toMatch(/^data:image\/png;base64,/);
  });

  it("skips non-image binary params", () => {
    expect(
      binaryThumbnailSrc("content_base64", "JVBERi0xLjQK", {
        mime_type: "application/pdf",
      }),
    ).toBeUndefined();
  });
});
