package dev.fragmentrpc.summary

import dev.fragmentrpc.graphql.GraphQLQuery
import dev.fragmentrpc.generated.*

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

@GraphQLQuery("""
query GetAndroidUserSummary(${'$'}id: ID!) {
  user(id: ${'$'}id) { id name }
}
""")
object AndroidUserSummaryQuery

@Composable
fun UserSummary(data: AndroidUserSummaryQueryData) {
    if (data.hasUser()) Surface(color = Color.White, shape = MaterialTheme.shapes.medium) {
        Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("UserService / ${data.user.id}", fontSize = 11.sp)
            Text(data.user.name, style = MaterialTheme.typography.titleLarge, modifier = Modifier.testTag("user-name"))
            Text("ID と名前だけを取得しました。", fontSize = 11.sp)
        }
    } else Text("ユーザーが見つかりません。", modifier = Modifier.testTag("missing-user"))
}
