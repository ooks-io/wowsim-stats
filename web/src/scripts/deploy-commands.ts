// discord command registration script
// run this to register slash commands with Discord

import { REST } from "@discordjs/rest";
import { Routes } from "discord-api-types/v10";
import { SlashCommandBuilder } from "@discordjs/builders";

// load environment variables
const DISCORD_APPLICATION_ID = process.env.DISCORD_APPLICATION_ID;
const DISCORD_BOT_TOKEN = process.env.DISCORD_BOT_TOKEN;

if (!DISCORD_APPLICATION_ID || !DISCORD_BOT_TOKEN) {
  console.error(
    "Error: DISCORD_APPLICATION_ID and DISCORD_BOT_TOKEN must be set",
  );
  console.error("Please set these environment variables and try again.");
  process.exit(1);
}

// define commands
const commands = [
  // /player command
  new SlashCommandBuilder()
    .setName("player")
    .setDescription("Look up a WoW Challenge Mode player")
    .addStringOption((option) =>
      option
        .setName("name")
        .setDescription("Player name")
        .setRequired(true)
        .setAutocomplete(true),
    )
    .addStringOption((option) =>
      option
        .setName("region")
        .setDescription("Region")
        .setRequired(true)
        .setAutocomplete(true),
    )
    .addStringOption((option) =>
      option
        .setName("realm")
        .setDescription("Realm")
        .setRequired(true)
        .setAutocomplete(true),
    ),

  // /leaderboard command
  new SlashCommandBuilder()
    .setName("leaderboard")
    .setDescription("View WoW Challenge Mode leaderboards")
    .addSubcommand((subcommand) =>
      subcommand
        .setName("dungeon")
        .setDescription("View dungeon leaderboard")
        .addStringOption((option) =>
          option
            .setName("dungeon")
            .setDescription("Choose a dungeon")
            .setRequired(true)
            .setAutocomplete(true),
        )
        .addStringOption((option) =>
          option
            .setName("scope")
            .setDescription("Leaderboard scope")
            .setRequired(true)
            .addChoices(
              { name: "Global", value: "global" },
              { name: "Region", value: "region" },
              { name: "Realm", value: "realm" },
            ),
        )
        .addStringOption((option) =>
          option
            .setName("region")
            .setDescription("Region (required for regional/realm scope)")
            .setRequired(false)
            .addChoices(
              { name: "US", value: "us" },
              { name: "EU", value: "eu" },
              { name: "KR", value: "kr" },
              { name: "TW", value: "tw" },
            ),
        )
        .addStringOption((option) =>
          option
            .setName("realm")
            .setDescription("Realm (required for realm scope)")
            .setRequired(false)
            .setAutocomplete(true),
        )
        .addIntegerOption((option) =>
          option
            .setName("limit")
            .setDescription("Number of runs to display (default: 10, max: 15)")
            .setRequired(false)
            .setMinValue(1)
            .setMaxValue(15),
        )
        .addStringOption((option) =>
          option
            .setName("season")
            .setDescription("Season (default: current season)")
            .setRequired(false),
        ),
    )
    .addSubcommand((subcommand) =>
      subcommand
        .setName("players")
        .setDescription("View player rankings")
        .addStringOption((option) =>
          option
            .setName("scope")
            .setDescription("Leaderboard scope")
            .setRequired(true)
            .addChoices(
              { name: "Global", value: "global" },
              { name: "Region", value: "region" },
              { name: "Realm", value: "realm" },
            ),
        )
        .addStringOption((option) =>
          option
            .setName("region")
            .setDescription("Region (required for regional/realm scope)")
            .setRequired(false)
            .addChoices(
              { name: "US", value: "us" },
              { name: "EU", value: "eu" },
              { name: "KR", value: "kr" },
              { name: "TW", value: "tw" },
            ),
        )
        .addStringOption((option) =>
          option
            .setName("realm")
            .setDescription("Realm (required for realm scope)")
            .setRequired(false)
            .setAutocomplete(true),
        )
        .addStringOption((option) =>
          option
            .setName("class")
            .setDescription("Filter by class")
            .setRequired(false)
            .setAutocomplete(true),
        )
        .addIntegerOption((option) =>
          option
            .setName("limit")
            .setDescription(
              "Number of players to display (default: 25, max: 25)",
            )
            .setRequired(false)
            .setMinValue(1)
            .setMaxValue(25),
        )
        .addStringOption((option) =>
          option
            .setName("season")
            .setDescription("Season (default: current season)")
            .setRequired(false),
        ),
    ),
].map((command) => command.toJSON());

// register commands with Discord
async function deployCommands() {
  console.log("Registering Discord slash commands...");
  console.log(`Application ID: ${DISCORD_APPLICATION_ID}`);
  console.log(`Commands to register: ${commands.length}`);

  const rest = new REST({ version: "10" }).setToken(DISCORD_BOT_TOKEN);

  try {
    console.log("Sending registration request to Discord...");

    const data = (await rest.put(
      Routes.applicationCommands(DISCORD_APPLICATION_ID),
      { body: commands },
    )) as any[];

    console.log(`✅ Successfully registered ${data.length} slash commands!`);
    console.log("\nRegistered commands:");
    data.forEach((cmd: any) => {
      console.log(`  - /${cmd.name}: ${cmd.description}`);
    });

    console.log("\nYou can now use these commands in Discord!");
    console.log(
      "Note: It may take a few minutes for commands to appear in all servers.",
    );
  } catch (error) {
    console.error("❌ Error registering commands:", error);
    process.exit(1);
  }
}

// run the deployment
deployCommands();
