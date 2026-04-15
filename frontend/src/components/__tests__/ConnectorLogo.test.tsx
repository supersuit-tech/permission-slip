import { render } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ConnectorLogo } from "../ConnectorLogo";

describe("ConnectorLogo SVG sanitization", () => {
  it("renders a trusted SVG without stripping safe elements", () => {
    const logo = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="#000"/></svg>';
    const { container } = render(<ConnectorLogo name="Test" logoSvg={logo} />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
    expect(container.querySelector("circle")).toBeInTheDocument();
  });

  it("strips <script> tags from malicious SVG input", () => {
    const malicious = '<svg xmlns="http://www.w3.org/2000/svg"><script>window.__pwned=true</script><circle cx="12" cy="12" r="10"/></svg>';
    const { container } = render(<ConnectorLogo name="Evil" logoSvg={malicious} />);
    expect(container.querySelector("script")).toBeNull();
    expect(container.querySelector("circle")).toBeInTheDocument();
  });

  it("strips inline event handlers like onload", () => {
    const malicious = '<svg xmlns="http://www.w3.org/2000/svg" onload="window.__pwned=true"><circle cx="12" cy="12" r="10"/></svg>';
    const { container } = render(<ConnectorLogo name="Evil" logoSvg={malicious} />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
    expect(svg?.getAttribute("onload")).toBeNull();
  });

  it("strips <foreignObject> to block HTML/script injection via SVG", () => {
    const malicious = '<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><iframe src="https://evil.example.com"></iframe></foreignObject></svg>';
    const { container } = render(<ConnectorLogo name="Evil" logoSvg={malicious} />);
    expect(container.querySelector("foreignObject")).toBeNull();
    expect(container.querySelector("iframe")).toBeNull();
  });

  it("strips javascript: URIs from href attributes", () => {
    const malicious = '<svg xmlns="http://www.w3.org/2000/svg"><a href="javascript:alert(1)"><circle cx="12" cy="12" r="10"/></a></svg>';
    const { container } = render(<ConnectorLogo name="Evil" logoSvg={malicious} />);
    const link = container.querySelector("a");
    // Either <a> is removed entirely or the href is stripped.
    if (link) {
      expect(link.getAttribute("href")).toBeNull();
    }
  });
});
