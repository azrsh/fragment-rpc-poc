import type { Avatar_user } from "../../../generated/fragments.js";

export function Avatar({ user }: { user: Avatar_user }) {
  return <img className="avatar" src={user.avatarUrl} alt="" width="76" height="76" />;
}
