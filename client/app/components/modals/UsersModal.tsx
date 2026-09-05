import { useQuery } from "@tanstack/react-query";
import { createUser, deleteUser, getUsers, type User } from "api/api";
import { AsyncButton } from "../AsyncButton";
import { useEffect, useState } from "react";
import { Trash } from "lucide-react";
import SubHeader from "../primitives/SubHeader";
import { useAppContext } from "~/providers/AppProvider";

export default function UsersModal() {
  const { user: currentUser } = useAppContext();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<"user" | "admin">("user");
  const [loading, setLoading] = useState(false);
  const [err, setError] = useState<string>();
  const [displayData, setDisplayData] = useState<User[]>([]);

  const { isPending, isError, data, error } = useQuery({
    queryKey: ["users"],
    queryFn: () => {
      return getUsers();
    },
  });

  useEffect(() => {
    if (data) {
      setDisplayData(data);
    }
  }, [data]);

  if (isError) {
    return <p className="error">Error: {error.message}</p>;
  }
  if (isPending) {
    return <p>Loading...</p>;
  }

  const handleCreateUser = () => {
    setError(undefined);
    if (username === "" || password === "") {
      setError("username and password are required");
      return;
    }
    setLoading(true);
    createUser(username, password, role)
      .then((r) => {
        setDisplayData([...displayData, r]);
        setUsername("");
        setPassword("");
        setRole("user");
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  };

  const handleDeleteUser = (id: number) => {
    setError(undefined);
    setLoading(true);
    deleteUser(id)
      .then((r) => {
        if (r.ok) {
          setDisplayData(displayData.filter((v) => v.id != id));
        } else {
          r.json().then((r) => setError(r.error));
        }
      })
      .finally(() => setLoading(false));
  };

  return (
    <div className="">
      <SubHeader>Users</SubHeader>
      <div className="flex flex-col gap-4">
        {displayData.map((v) => (
          <div key={v.id} className="flex gap-2">
            <div className="bg p-3 rounded-md flex-grow">
              {v.username}{" "}
              <span className="text-sm opacity-60">({v.role})</span>
            </div>
            {v.id !== currentUser?.id && (
              <AsyncButton
                loading={loading}
                onClick={() => handleDeleteUser(v.id)}
                confirm
                danger
              >
                <Trash size={16} />
              </AsyncButton>
            )}
          </div>
        ))}
        <div className="flex flex-col gap-2 w-3/5">
          <input
            type="text"
            placeholder="Username"
            className="mx-auto fg bg rounded-md p-3 flex-grow w-full"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
          <input
            type="password"
            placeholder="Password"
            className="mx-auto fg bg rounded-md p-3 flex-grow w-full"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <select
            className="fg bg rounded-md p-3 w-full"
            value={role}
            onChange={(e) => setRole(e.target.value as "user" | "admin")}
          >
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
          <AsyncButton loading={loading} onClick={handleCreateUser}>
            Create User
          </AsyncButton>
        </div>
        {err && <p className="error">{err}</p>}
      </div>
    </div>
  );
}
