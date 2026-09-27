import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "./api/client";
import type { components } from "./gen/api";

type VersionInfo = components["schemas"]["VersionInfo"];
type State = { kind: "loading" } | { kind: "ok"; info: VersionInfo } | { kind: "error" };

export default function App() {
  const { t, i18n } = useTranslation();
  const [state, setState] = useState<State>({ kind: "loading" });

  useEffect(() => {
    let active = true;
    api
      .GET("/version")
      .then(({ data }) => {
        if (active) setState(data ? { kind: "ok", info: data } : { kind: "error" });
      })
      .catch(() => active && setState({ kind: "error" }));
    return () => {
      active = false;
    };
  }, []);

  const toggleLanguage = () => void i18n.changeLanguage(i18n.language === "de" ? "en" : "de");

  return (
    <main style={{ fontFamily: "system-ui, sans-serif", maxWidth: 640, margin: "3rem auto" }}>
      <h1>{t("app.title")}</h1>
      <p>{t("app.phaseNotice")}</p>
      <section aria-label={t("app.master")}>
        {state.kind === "loading" && <p>{t("app.loading")}</p>}
        {state.kind === "error" && <p role="alert">{t("app.unreachable")}</p>}
        {state.kind === "ok" && (
          <dl>
            <dt>{t("app.version")}</dt>
            <dd>{state.info.version}</dd>
            <dt>{t("app.protocol")}</dt>
            <dd>{state.info.protocolVersion}</dd>
            <dt>{t("app.schema")}</dt>
            <dd>{state.info.schemaVersion}</dd>
          </dl>
        )}
      </section>
      <button type="button" onClick={toggleLanguage}>
        {t("language.switch")}
      </button>
    </main>
  );
}
