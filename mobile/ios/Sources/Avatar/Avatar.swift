import SwiftUI
import GraphQLDocuments

@GraphQLFragment("""
    fragment IosAvatar_user on User {
      avatarUrl
    }
    """)
enum IosAvatarFragment {}


struct IosAvatar: View {
    let user: IosAvatarFragment.Data
    var body: some View {
        VStack(spacing: 5) {
            Image(systemName: "person.crop.square.fill")
                .font(.system(size: 54)).foregroundStyle(.indigo)
            Text(user.avatarUrl).font(.system(size: 9, design: .monospaced))
                .foregroundStyle(.secondary).accessibilityIdentifier("avatar-url")
        }
    }
}
