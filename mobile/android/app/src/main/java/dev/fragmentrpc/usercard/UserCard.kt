package dev.fragmentrpc.usercard

import dev.fragmentrpc.graphql.GraphQLFragment
import dev.fragmentrpc.generated.*

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import dev.fragmentrpc.avatar.Avatar
import dev.fragmentrpc.organization.OrganizationBadge

@GraphQLFragment("""
fragment AndroidUserCard_user on User {
  id
  name
  ...AndroidAvatar_user
  organization { ...AndroidOrganizationBadge_organization }
}
""")
object AndroidUserCardFragment

@Composable
fun UserCard(user: AndroidUserCardFragmentData) {
    Surface(color = Color.White, shape = MaterialTheme.shapes.medium) {
        Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Row(horizontalArrangement = Arrangement.spacedBy(18.dp)) {
                Avatar(user.asAndroidAvatarFragmentData())
                Column {
                    Text("UserService / ${user.id}", fontSize = 11.sp, color = Color(0xFF64738B))
                    Text(user.name, style = MaterialTheme.typography.titleLarge, modifier = Modifier.testTag("user-name"))
                }
            }
            if (user.organization != null) OrganizationBadge(user.organization.asAndroidOrganizationBadgeFragmentData())
            else Text("所属組織はありません。", modifier = Modifier.testTag("no-organization"))
        }
    }
}
