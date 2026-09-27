import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import i18n from "./i18n";

function mockFetch(response: Response | Error) {
  vi.stubGlobal(
    "fetch",
    vi.fn(() => (response instanceof Error ? Promise.reject(response) : Promise.resolve(response))),
  );
}

describe("App", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    void i18n.changeLanguage("de");
  });

  it("shows master version information in German by default", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          version: "1.2.3",
          commit: "abc",
          buildDate: "today",
          protocolVersion: 1,
          schemaVersion: 3,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    render(<App />);
    expect(await screen.findByText("1.2.3")).toBeInTheDocument();
    expect(screen.getByText("Protokollversion")).toBeInTheDocument();
  });

  it("shows an error when the master is unreachable", async () => {
    mockFetch(new Error("network"));
    render(<App />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Master nicht erreichbar");
  });
});
