package dev.fragmentrpc.organization

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

@GraphQLFragment("""
fragment AndroidOrganizationBadge_organization on Organization {
  name
  website
}
""")
object AndroidOrganizationBadgeFragment

@Composable
fun OrganizationBadge(organization: AndroidOrganizationBadgeFragmentData) {
    Surface(color = Color(0xFFF0F3FA), shape = MaterialTheme.shapes.small) {
        Column(Modifier.fillMaxWidth().padding(14.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text("OrganizationService", fontSize = 10.sp, color = Color(0xFF64738B))
            Text(organization.name, style = MaterialTheme.typography.titleMedium, modifier = Modifier.testTag("organization-name"))
            Text(organization.website, fontSize = 10.sp)
        }
    }
}
