import type { OrganizationBadge_organization } from "../../../generated/fragments.js";

export function OrganizationBadge({ organization }: { organization: OrganizationBadge_organization }) {
  return <div className="organization">
    <span className="organization-icon" aria-hidden="true">↗</span>
    <div><small>OrganizationService</small><a href={organization.website} target="_blank" rel="noreferrer">{organization.name}</a></div>
  </div>;
}
