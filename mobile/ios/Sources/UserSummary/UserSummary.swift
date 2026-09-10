import SwiftUI
import GraphQLDocuments

@GraphQLQuery("""
    query GetIosUserSummary($id: ID!) {
      user(id: $id) { id name }
    }
    """)
enum IosUserSummaryQuery {}


struct IosUserSummary: View {
    let data: IosUserSummaryQuery.Data
    var body: some View {
        if data.hasUser {
            VStack(alignment: .leading, spacing: 10) {
                Text("UserService / \(data.user.id)").font(.caption).foregroundStyle(.secondary)
                Text(data.user.name).font(.title2.bold()).accessibilityIdentifier("user-name")
                Text("ID と名前だけを取得しました。").font(.caption)
            }.padding().frame(maxWidth: .infinity, alignment: .leading)
                .background(.white).clipShape(RoundedRectangle(cornerRadius: 12))
        } else { Text("ユーザーが見つかりません。").accessibilityIdentifier("missing-user") }
    }
}
