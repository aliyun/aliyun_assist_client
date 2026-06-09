/** A single unit of user input in a turn. */
export type UserInput = {
  type: "text";
  text: string;
};

/** Parameters for starting or resuming a thread/session. */
export type ThreadStartParams = {
  cwd?: string;
  ephemeral?: boolean;
  modelProvider?: string;
};

/** Response from threadStart — contains the thread id. */
export type ThreadStartResponse = {
  threadId: string;
};

/** Parameters for starting a turn within an existing thread. */
export type TurnStartParams = {
  threadId: string;
  input: UserInput[];
};
