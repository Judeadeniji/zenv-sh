import { sql } from "drizzle-orm";
import { db } from "./db.js";

/**
 * Map Better Auth roles to zEnv organization_members roles.
 * BA: owner, admin, member
 * zEnv: admin, senior_dev, dev, contractor, ci_bot
 */
function mapRole(baRole: string): string {
  switch (baRole) {
    case "owner":
    case "admin":
      return "admin";
    case "member":
    default:
      return "dev";
  }
}

/**
 * Resolve a Better Auth user ID to a zEnv user UUID.
 * Returns null if the user hasn't set up their vault yet.
 */
async function resolveZenvUser(
  baUserId: string,
): Promise<string | null> {
  const rows = await db.execute(
    sql`SELECT id::text FROM users WHERE better_auth_user_id = ${baUserId} LIMIT 1`,
  );
  if (rows.length === 0) return null;
  return rows[0].id as string;
}

/**
 * After a BA organization is created, insert a corresponding row in
 * zEnv's organizations table and add the creator as an admin member.
 */
export async function syncOrgToZenv(
  org: { id: string; name: string },
  member: { userId: string },
): Promise<void> {
  const zenvUserId = await resolveZenvUser(member.userId);
  if (!zenvUserId) {
    console.warn(
      `[hooks] syncOrgToZenv: BA user ${member.userId} has no zEnv user (vault not set up). Skipping org sync.`,
    );
    return;
  }

  try {
    await db.execute(
      sql`INSERT INTO organizations (name, owner_id, better_auth_org_id)
          VALUES (${org.name}, ${zenvUserId}::uuid, ${org.id})
          ON CONFLICT (better_auth_org_id) DO NOTHING`,
    );
  } catch (err) {
    console.error(`[hooks] syncOrgToZenv failed:`, err);
  }
}

/**
 * After a BA member is added to an org, insert into zEnv's organization_members table.
 */
export async function syncMemberToZenv(
  member: { userId: string; role: string },
  org: { id: string },
): Promise<void> {
  const zenvUserId = await resolveZenvUser(member.userId);
  if (!zenvUserId) {
    console.warn(
      `[hooks] syncMemberToZenv: BA user ${member.userId} has no zEnv user. Skipping member sync.`,
    );
    return;
  }

  const zenvRole = mapRole(member.role);

  try {
    // Resolve zEnv org ID from BA org ID
    const orgRows = await db.execute(
      sql`SELECT id::text FROM organizations WHERE better_auth_org_id = ${org.id} LIMIT 1`,
    );
    if (orgRows.length === 0) {
      console.warn(
        `[hooks] syncMemberToZenv: no zEnv org for BA org ${org.id}. Skipping.`,
      );
      return;
    }
    const zenvOrgId = orgRows[0].id as string;

    await db.execute(
      sql`INSERT INTO organization_members (organization_id, user_id, role)
          VALUES (${zenvOrgId}::uuid, ${zenvUserId}::uuid, ${zenvRole})
          ON CONFLICT (organization_id, user_id) DO UPDATE SET role = ${zenvRole}`,
    );
  } catch (err) {
    console.error(`[hooks] syncMemberToZenv failed:`, err);
  }
}
