import type { GetUserSummaryResponse } from "../../../generated/app_pb.js";

export function UserSummary({ data }: { data: GetUserSummaryResponse }) {
  return data.user ? <article className="user-card summary"><small>UserService / {data.user.id}</small><h2>{data.user.name}</h2><p>この operation は名前と ID だけを取得します。</p></article> : <p className="empty-result">ユーザーが見つかりません。</p>;
}
