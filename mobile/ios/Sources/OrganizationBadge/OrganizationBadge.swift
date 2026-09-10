import SwiftUI
import GraphQLDocuments

@GraphQLFragment("""
    fragment IosOrganizationBadge_organization on Organization {
      name
      website
    }
    """)
enum IosOrganizationBadgeFragment {}


struct IosOrganizationBadge: View {
    let organization: IosOrganizationBadgeFragment.Data
    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Label("OrganizationService", systemImage: "building.2").font(.caption).foregroundStyle(.secondary)
            Text(organization.name).font(.headline).accessibilityIdentifier("organization-name")
            Text(organization.website).font(.caption).foregroundStyle(.secondary)
        }.frame(maxWidth: .infinity, alignment: .leading).padding()
            .background(Color.indigo.opacity(0.06)).clipShape(RoundedRectangle(cornerRadius: 8))
    }
}
