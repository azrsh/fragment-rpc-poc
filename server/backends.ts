import { UserService, OrganizationService } from "../generated/backend_pb.js";
import { serveGrpc } from "./network.js";

const users = [
  { id: "u1", name: "Aki Tanaka", email: "aki@example.test", avatarUrl: "/avatars/aki.svg", organizationId: "o1" },
  { id: "u2", name: "Ren Sato", email: "ren@example.test", avatarUrl: "/avatars/ren.svg", organizationId: "o2" },
  { id: "u3", name: "Mika Ito", email: "mika@example.test", avatarUrl: "/avatars/mika.svg", organizationId: "" },
];
const organizations = [
  { id: "o1", name: "Northstar Studio", website: "https://example.com/northstar" },
  { id: "o2", name: "Fieldwork Labs", website: "https://example.com/fieldwork" },
];

export async function startBackends(onCall: (service: string, id: string) => void = () => {}) {
  const user = await serveGrpc(router => router.service(UserService, {
    getUser(request) {
      onCall("UserService.GetUser", request.id);
      return { user: users.find(u => u.id === request.id) };
    },
  }));
  try {
    const organization = await serveGrpc(router => router.service(OrganizationService, {
      getOrganization(request) {
        onCall("OrganizationService.GetOrganization", request.id);
        return { organization: organizations.find(o => o.id === request.id) };
      },
    }));
    return { user, organization, close: async () => { await Promise.all([user.close(), organization.close()]); } };
  } catch (error) { await user.close(); throw error; }
}
