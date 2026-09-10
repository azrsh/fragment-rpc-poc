package dev.fragmentrpc.userpage

import dev.fragmentrpc.graphql.GraphQLQuery
import dev.fragmentrpc.generated.*

import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import dev.fragmentrpc.usercard.UserCard

@GraphQLQuery("""
query GetAndroidUserPage(${'$'}id: ID!) {
  user(id: ${'$'}id) { ...AndroidUserCard_user }
}
""")
object AndroidUserPageQuery

@Composable
fun UserPage(data: AndroidUserPageQueryData) {
    if (data.hasUser()) UserCard(data.user.asAndroidUserCardFragmentData())
    else Text("ユーザーが見つかりません。", modifier = Modifier.testTag("missing-user"))
}
