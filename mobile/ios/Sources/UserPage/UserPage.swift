import SwiftUI
import GraphQLDocuments

@GraphQLQuery("""
    query GetIosUserPage($id: ID!) {
      user(id: $id) { ...IosUserCard_user }
    }
    """)
enum IosUserPageQuery {}


struct IosUserPage: View {
    let data: IosUserPageQuery.Data
    var body: some View {
        if data.hasUser { IosUserCard(user: data.user.asIosUserCardFragmentData()) }
        else { Text("ユーザーが見つかりません。").accessibilityIdentifier("missing-user") }
    }
}
