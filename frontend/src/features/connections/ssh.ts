export const SSH_AUTH_KEY = "key";
export const SSH_AUTH_PASSWORD = "password";
export const SSH_AUTH_AGENT = "agent";

export const SSH_DEFAULTS: Record<string, string> = {
  sshEnabled: "false",
  sshHost: "",
  sshPort: "22",
  sshUser: "",
  sshAuth: SSH_AUTH_KEY,
  sshKeyPath: "",
  sshPassword: "",
  sshPassphrase: "",
};

export const SSH_PARAM_KEYS = Object.keys(SSH_DEFAULTS);

export function copySshParams(source: Record<string, string>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const key of SSH_PARAM_KEYS) {
    out[key] = source[key] ?? SSH_DEFAULTS[key];
  }
  return out;
}
