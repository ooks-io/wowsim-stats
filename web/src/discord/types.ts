// discord interaction types

export interface DiscordInteraction {
  id: string;
  application_id: string;
  type: InteractionType;
  data?: InteractionData;
  guild_id?: string;
  channel_id?: string;
  member?: GuildMember;
  user?: User;
  token: string;
  version: number;
  message?: Message;
}

export enum InteractionType {
  PING = 1,
  APPLICATION_COMMAND = 2,
  MESSAGE_COMPONENT = 3,
  APPLICATION_COMMAND_AUTOCOMPLETE = 4,
}

export interface InteractionData {
  id?: string;
  name?: string;
  type?: number;
  resolved?: unknown;
  options?: CommandOption[];
  custom_id?: string;
  component_type?: number;
  values?: string[];
  target_id?: string;
}

export interface CommandOption {
  name: string;
  type: number;
  value?: string | number | boolean;
  options?: CommandOption[];
  focused?: boolean;
}

export interface User {
  id: string;
  username: string;
  discriminator: string;
  avatar?: string;
  bot?: boolean;
}

export interface GuildMember {
  user?: User;
  nick?: string;
  roles: string[];
  joined_at: string;
  deaf: boolean;
  mute: boolean;
}

export interface Message {
  id: string;
  type: number;
  content: string;
  channel_id: string;
  author: User;
  embeds?: Embed[];
  components?: MessageComponent[];
}

export interface Embed {
  title?: string;
  description?: string;
  url?: string;
  color?: number;
  fields?: EmbedField[];
  author?: EmbedAuthor;
  footer?: EmbedFooter;
  thumbnail?: EmbedImage;
  image?: EmbedImage;
  timestamp?: string;
}

export interface EmbedField {
  name: string;
  value: string;
  inline?: boolean;
}

export interface EmbedAuthor {
  name: string;
  url?: string;
  icon_url?: string;
}

export interface EmbedFooter {
  text: string;
  icon_url?: string;
}

export interface EmbedImage {
  url: string;
  proxy_url?: string;
  height?: number;
  width?: number;
}

export interface MessageComponent {
  type: ComponentType;
  custom_id?: string;
  disabled?: boolean;
  style?: ButtonStyle;
  label?: string;
  emoji?: {
    id?: string;
    name?: string;
    animated?: boolean;
  };
  url?: string;
  options?: SelectOption[];
  placeholder?: string;
  min_values?: number;
  max_values?: number;
  components?: MessageComponent[];
}

export enum ComponentType {
  ACTION_ROW = 1,
  BUTTON = 2,
  STRING_SELECT = 3,
  TEXT_INPUT = 4,
  USER_SELECT = 5,
  ROLE_SELECT = 6,
  MENTIONABLE_SELECT = 7,
  CHANNEL_SELECT = 8,
}

export enum ButtonStyle {
  PRIMARY = 1,
  SECONDARY = 2,
  SUCCESS = 3,
  DANGER = 4,
  LINK = 5,
}

export interface SelectOption {
  label: string;
  value: string;
  description?: string;
  emoji?: {
    id?: string;
    name?: string;
    animated?: boolean;
  };
  default?: boolean;
}

export interface InteractionResponse {
  type: InteractionResponseType;
  data?: InteractionResponseData;
}

export enum InteractionResponseType {
  PONG = 1,
  CHANNEL_MESSAGE_WITH_SOURCE = 4,
  DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE = 5,
  DEFERRED_UPDATE_MESSAGE = 6,
  UPDATE_MESSAGE = 7,
  APPLICATION_COMMAND_AUTOCOMPLETE_RESULT = 8,
}

export interface InteractionResponseData {
  tts?: boolean;
  content?: string;
  embeds?: Embed[];
  allowed_mentions?: unknown;
  flags?: number;
  components?: MessageComponent[];
  choices?: AutocompleteChoice[];
}

export interface AutocompleteChoice {
  name: string;
  value: string | number;
}

// custom types for command handling

export interface LeaderboardCommandOptions {
  dungeon?: string;
  scope: "global" | "region" | "realm";
  region?: "us" | "eu" | "kr" | "tw";
  realm?: string;
  season?: string;
  class?: string;
  page?: number;
}

export interface PlayerCommandOptions {
  name: string;
  region: "us" | "eu" | "kr" | "tw";
  realm: string;
  season?: string;
}

export interface PaginationData {
  command: string;
  subcommand?: string;
  page: number;
  totalPages?: number;
  [key: string]: unknown;
}
