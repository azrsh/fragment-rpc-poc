import { useState, type FormEvent } from "react";
import { createRoot } from "react-dom/client";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { toJson } from "@bufbuild/protobuf";
import { AppService, GetUserPageResponseSchema, GetUserSummaryResponseSchema, type GetUserPageResponse, type GetUserSummaryResponse } from "../generated/app_pb.js";
import { UserPage } from "./pages/UserPage/UserPage.js";
import { UserSummary } from "./pages/UserSummary/UserSummary.js";
import contract from "../generated/app.proto?raw";
import fragment from "./components/UserCard/UserCard.graphql?raw";
import "./style.css";

const client = createClient(AppService, createConnectTransport({ baseUrl: window.location.origin, useBinaryFormat: true }));
type Result = { kind: "page"; data: GetUserPageResponse } | { kind: "summary"; data: GetUserSummaryResponse };

function App() {
  const [id, setId] = useState("u1");
  const [operation, setOperation] = useState<"page" | "summary">("page");
  const [result, setResult] = useState<Result>();
  const [payload, setPayload] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const [elapsed, setElapsed] = useState(0);
  const [tab, setTab] = useState<"response" | "contract" | "fragment">("response");
  const [lastMethod, setLastMethod] = useState("");

  async function fetchUser(event: FormEvent) {
    event.preventDefault();
    setPending(true); setError(""); setResult(undefined); setPayload("");
    const start = performance.now();
    const method = operation === "page" ? "GetUserPage" : "GetUserSummary";
    setLastMethod(method);
    try {
      if (operation === "page") {
        const data = await client.getUserPage({ id }, { timeoutMs: 5000 });
        setResult({ kind: "page", data });
        setPayload(JSON.stringify(toJson(GetUserPageResponseSchema, data), null, 2));
      } else {
        const data = await client.getUserSummary({ id }, { timeoutMs: 5000 });
        setResult({ kind: "summary", data });
        setPayload(JSON.stringify(toJson(GetUserSummaryResponseSchema, data), null, 2));
      }
      setElapsed(Math.round(performance.now() - start));
      setTab("response");
    } catch (e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setPending(false); }
  }

  return <main>
    <header><a className="brand" href="/" aria-label="Fragment RPC ホーム"><span className="brand-symbol" aria-hidden="true">{`{↗}`}</span> Fragment RPC</a><span className="poc">Local proof of concept</span></header>
    <section className="intro"><h1>画面のデータ定義が、<br />そのまま RPC になる。</h1><p>コンポーネントに置いたフラグメントをビルド時に合成。<br />生成した専用 API で、2 つの gRPC サービスをつなぎます。</p></section>
    <ol className="pipeline" aria-label="生成と実行の流れ"><li><strong>Fragment</strong><span>コンポーネントの隣で定義</span></li><li><strong>Compile</strong><span>合成・検証・型生成</span></li><li><strong>Protobuf API</strong><span>operation ごとの専用 RPC</span></li><li><strong>gRPC services</strong><span>ユーザーと所属組織を取得</span></li></ol>
    <div className="workspace">
      <section className="request-panel" aria-labelledby="request-title"><div className="panel-title"><h2 id="request-title">API を試す</h2><span>Connect / Protobuf</span></div>
        <form onSubmit={fetchUser}><fieldset disabled={pending}><legend>取得するデータ</legend><div className="operation-options"><label className={operation === "page" ? "selected" : ""}><input type="radio" name="operation" value="page" checked={operation === "page"} onChange={() => setOperation("page")} /><strong>プロフィール</strong><small>名前・アバター・所属組織</small></label><label className={operation === "summary" ? "selected" : ""}><input type="radio" name="operation" value="summary" checked={operation === "summary"} onChange={() => setOperation("summary")} /><strong>サマリー</strong><small>ID と名前だけ</small></label></div>
          <label className="select-label" htmlFor="user-id">ユーザー</label><select id="user-id" value={id} onChange={e => setId(e.target.value)}><option value="u1">u1 — Aki Tanaka</option><option value="u2">u2 — Ren Sato</option><option value="u3">u3 — Mika Ito（組織なし）</option><option value="missing">missing — 存在しないユーザー</option></select>
          <button className="fetch-button" type="submit">{pending ? "取得しています…" : "生成 API を呼び出す"}</button></fieldset></form>
        <div className="result-heading"><h3>コンポーネントの表示</h3>{result && <span>{elapsed} ms</span>}</div>
        <div aria-live="polite" aria-busy={pending}>{error ? <p role="alert" className="error">{error}</p> : result ? result.kind === "page" ? <UserPage data={result.data} /> : <UserSummary data={result.data} /> : <div className="placeholder"><span aria-hidden="true">{`{ }`}</span><p>API を呼び出すと、取得したデータが<br />コンポーネントに渡されます。</p></div>}</div>
      </section>
      <section className="artifact-panel" aria-label="API の成果物"><div className="tabs" role="tablist" aria-label="成果物"><button id="tab-response" role="tab" aria-selected={tab === "response"} aria-controls="artifact" onClick={() => setTab("response")}>レスポンス</button><button id="tab-contract" role="tab" aria-selected={tab === "contract"} aria-controls="artifact" onClick={() => setTab("contract")}>生成された .proto</button><button id="tab-fragment" role="tab" aria-selected={tab === "fragment"} aria-controls="artifact" onClick={() => setTab("fragment")}>フラグメント</button></div>
        <div className="artifact-caption">{tab === "response" ? lastMethod ? `/app.v1.AppService/${lastMethod}` : "RPC の実行結果を JSON 表示" : tab === "contract" ? "generated/app.proto" : "app/components/UserCard/UserCard.graphql"}</div>
        <pre id="artifact" role="tabpanel" aria-labelledby={`tab-${tab}`} tabIndex={0}><code>{tab === "contract" ? contract : tab === "fragment" ? fragment : payload || (error ? error : "// まだ API は呼び出されていません。")}</code></pre>
        <p className="artifact-note">{tab === "response" ? "送るのは変数だけ。選択していない email は、生成型にもレスポンスにも含まれません。" : tab === "contract" ? "入力は variables、出力は selection set。通常の Protobuf ツールで別言語のクライアントも生成できます。" : "子コンポーネントのフラグメントを参照。変更後は npm run build とサーバーの再起動で API に反映します。"}</p>
      </section>
    </div><footer><span>GraphQL はビルド時の DSL。実行時は生成済み API と実行計画。</span><code>npm run demo</code><span>で gRPC を直接検証できます。</span></footer>
  </main>;
}
createRoot(document.getElementById("root")!).render(<App />);
