import { createClient } from "@connectrpc/connect";
import { createGrpcTransport } from "@connectrpc/connect-node";
import { UserService, OrganizationService } from "../generated/backend_pb.js";
import type { ExecutionContext, Resolvers } from "./execute.js";

function once(context: ExecutionContext, key: string, load: () => Promise<unknown>) {
  let value = context.cache.get(key);
  if (!value) { value = load(); context.cache.set(key, value); }
  return value;
}

export function createResolvers(userUrl: string, organizationUrl: string): Resolvers {
  const users = createClient(UserService, createGrpcTransport({ baseUrl: userUrl }));
  const organizations = createClient(OrganizationService, createGrpcTransport({ baseUrl: organizationUrl }));
  return {
    "Query.user": (_source, args, context) => once(context, `user:${args.id}`, async () => {
      const response = await users.getUser({ id: String(args.id) }, { signal: context.signal, timeoutMs: 2000 });
      return response.user;
    }),
    "User.organization": (source, _args, context) => {
      if (!source.organizationId) return undefined;
      return once(context, `organization:${source.organizationId}`, async () => {
        const response = await organizations.getOrganization({ id: String(source.organizationId) }, { signal: context.signal, timeoutMs: 2000 });
        return response.organization;
      });
    },
  };
}
