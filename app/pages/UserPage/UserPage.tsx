import type { GetUserPageResponse } from "../../../generated/app_pb.js";
import { UserCard } from "../../components/UserCard/UserCard.js";

export function UserPage({ data }: { data: GetUserPageResponse }) {
  return data.user ? <UserCard user={data.user} /> : <p className="empty-result">ユーザーが見つかりません。</p>;
}
