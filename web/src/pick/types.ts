export type Repo = {
  id: string;
  name: string;
  description: string;
  language: string;
  pushedAt: string;
  archived: boolean;
  // blocked is why this repo cannot be added; empty when it can.
  blocked: string;
};

export type ReposResponse = {
  token: string;
  org: string;
  orgs: string[];
  authenticated: boolean;
  error: string;
  repos: Repo[];
};

export type AddResult = {
  added: { name: string; url: string; branch: string }[];
  refused: { id: string; reason: string }[];
};
