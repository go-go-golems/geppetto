const gp = require("geppetto");

const registries = gp.profiles.listRegistries();
assert(registries.length === 1, "expected single sqlite-backed registry");
assert(registries[0].slug === "workspace-db", "expected workspace-db registry");

const created = gp.profiles.createProfile(
  {
    slug: "ops",
    display_name: "Ops",
    description: "Operations profile",
    stack: [{ profile_slug: "default" }],
    runtime: {
      system_prompt: "You are operations support."
    }
  },
  {
    registrySlug: "workspace-db",
    write: { actor: "example-script", source: "15_profiles_crud_sqlite.js" }
  }
);
assert(created.slug === "ops", "createProfile should return ops profile");

const updated = gp.profiles.updateProfile(
  "ops",
  { description: "Operations profile updated" },
  {
    registrySlug: "workspace-db",
    write: { actor: "example-script", source: "15_profiles_crud_sqlite.js" }
  }
);
assert(updated.description === "Operations profile updated", "updateProfile should persist description");

gp.profiles.setDefaultProfile("ops", {
  registrySlug: "workspace-db",
  write: { actor: "example-script", source: "15_profiles_crud_sqlite.js" }
});
const resolvedDefault = gp.profiles.resolve({ registrySlug: "workspace-db" });
assert(resolvedDefault.profileSlug === "ops", "resolve() should use updated default profile");

gp.profiles.setDefaultProfile("default", {
  registrySlug: "workspace-db",
  write: { actor: "example-script", source: "15_profiles_crud_sqlite.js" }
});

gp.profiles.deleteProfile("ops", {
  registrySlug: "workspace-db",
  write: { actor: "example-script", source: "15_profiles_crud_sqlite.js" }
});

let deleted = false;
try {
  gp.profiles.getProfile("ops", "workspace-db");
} catch (e) {
  deleted = /profile not found/i.test(String(e));
}
assert(deleted, "ops profile should be deleted");

console.log("sqlite CRUD checks: PASS");
