import { useEffect, useState } from "react";
import { Loader2, CheckCircle, Trash2, Shield, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { adminApi, type AdminUser } from "@/services/api";

export default function AdminUsers() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [actionLoading, setActionLoading] = useState<number | null>(null);

  const fetchUsers = async () => {
    try {
      const data = await adminApi.listUsers();
      setUsers(data || []);
    } catch (err: any) {
      setError(err.message || "Failed to fetch users");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleVerify = async (id: number) => {
    setActionLoading(id);
    try {
      await adminApi.verifyUser(id);
      setUsers((prev) =>
        prev.map((u) => (u.id === id ? { ...u, is_verified: true } : u))
      );
    } catch (err: any) {
      setError(err.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm("Are you sure? This will delete the user and all their contacts.")) return;
    setActionLoading(id);
    try {
      await adminApi.deleteUser(id);
      setUsers((prev) => prev.filter((u) => u.id !== id));
    } catch (err: any) {
      setError(err.message);
    } finally {
      setActionLoading(null);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Shield className="h-6 w-6 text-primary" />
        <h1 className="text-2xl font-bold tracking-tight">Admin — User Management</h1>
      </div>

      {error && (
        <div className="rounded-lg border border-danger-muted bg-danger-muted px-4 py-3 text-sm text-danger">
          {error}
        </div>
      )}

      <div className="rounded-xl border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50">
              <th className="px-4 py-3 text-left font-medium">ID</th>
              <th className="px-4 py-3 text-left font-medium">Name</th>
              <th className="px-4 py-3 text-left font-medium">Email</th>
              <th className="px-4 py-3 text-left font-medium">Verified</th>
              <th className="px-4 py-3 text-left font-medium">Role</th>
              <th className="px-4 py-3 text-left font-medium">Created</th>
              <th className="px-4 py-3 text-right font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id} className="border-b border-border last:border-0 hover:bg-muted/20 transition-colors">
                <td className="px-4 py-3 text-muted-foreground">{user.id}</td>
                <td className="px-4 py-3 font-medium">{user.name}</td>
                <td className="px-4 py-3 text-muted-foreground">{user.email}</td>
                <td className="px-4 py-3">
                  {user.is_verified ? (
                    <span className="inline-flex items-center gap-1 text-success text-xs font-medium">
                      <CheckCircle className="h-3.5 w-3.5" /> Verified
                    </span>
                  ) : (
                    <span className="text-xs text-warning font-medium">Pending</span>
                  )}
                </td>
                <td className="px-4 py-3">
                  {user.role === "admin" ? (
                    <span className="inline-flex items-center gap-1 text-primary text-xs font-medium">
                      <ShieldCheck className="h-3.5 w-3.5" /> Admin
                    </span>
                  ) : (
                    <span className="text-xs text-muted-foreground">User</span>
                  )}
                </td>
                <td className="px-4 py-3 text-muted-foreground text-xs">
                  {new Date(user.created_at).toLocaleDateString()}
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="flex items-center justify-end gap-2">
                    {!user.is_verified && (
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={actionLoading === user.id}
                        onClick={() => handleVerify(user.id)}
                        className="text-xs"
                      >
                        {actionLoading === user.id ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          "Verify"
                        )}
                      </Button>
                    )}
                    {user.role !== "admin" && (
                      <Button
                        size="sm"
                        variant="destructive"
                        disabled={actionLoading === user.id}
                        onClick={() => handleDelete(user.id)}
                        className="text-xs"
                      >
                        {actionLoading === user.id ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <Trash2 className="h-3.5 w-3.5" />
                        )}
                      </Button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
            {users.length === 0 && (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                  No users found
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
