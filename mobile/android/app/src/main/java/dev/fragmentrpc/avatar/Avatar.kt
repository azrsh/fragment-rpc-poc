package dev.fragmentrpc.avatar

import dev.fragmentrpc.graphql.GraphQLFragment
import dev.fragmentrpc.generated.*

import androidx.compose.foundation.layout.Column
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.sp

@GraphQLFragment("""
fragment AndroidAvatar_user on User {
  avatarUrl
}
""")
object AndroidAvatarFragment

@Composable
fun Avatar(user: AndroidAvatarFragmentData) {
    Column {
        Text("◉", fontSize = 42.sp, color = Color(0xFF4961C5))
        Text(user.avatarUrl, fontSize = 9.sp, color = Color(0xFF64738B))
    }
}
