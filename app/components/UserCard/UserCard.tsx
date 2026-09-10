import type { UserCard_user } from "../../../generated/fragments.js";
import { Avatar } from "../Avatar/Avatar.js";
import { OrganizationBadge } from "../OrganizationBadge/OrganizationBadge.js";

export function UserCard({ user }: { user: UserCard_user }) {
  return <article className="user-card">
    <div className="identity"><Avatar user={user} /><div><small>UserService / {user.id}</small><h2>{user.name}</h2></div></div>
    {user.organization ? <OrganizationBadge organization={user.organization} /> : <p className="empty-organization">所属組織はありません。</p>}
  </article>;
}
