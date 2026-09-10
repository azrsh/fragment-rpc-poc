import SwiftUI
import GraphQLDocuments

@GraphQLFragment("""
    fragment IosUserCard_user on User {
      id
      name
      ...IosAvatar_user
      organization { ...IosOrganizationBadge_organization }
    }
    """)
enum IosUserCardFragment {}


struct IosUserCard: View {
    let user: IosUserCardFragment.Data
    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack(spacing: 18) {
                IosAvatar(user: user.asIosAvatarFragmentData())
                VStack(alignment: .leading) {
                    Text("UserService / \(user.id)").font(.caption).foregroundStyle(.secondary)
                    Text(user.name).font(.title2.bold()).accessibilityIdentifier("user-name")
                }
            }
            if let organization = user.organization {
                IosOrganizationBadge(organization: organization.asIosOrganizationBadgeFragmentData())
            } else {
                Text("所属組織はありません。").accessibilityIdentifier("no-organization")
            }
        }.padding().frame(maxWidth: .infinity, alignment: .leading)
            .background(.white).clipShape(RoundedRectangle(cornerRadius: 12))
    }
}
