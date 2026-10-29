import { AssistantPage } from "~/api/assistant/schema";

// Questions offered on an empty conversation, each answerable with the
// read-only tools, picked for the page on screen.
const SUGGESTIONS = {
  repository: [
    "What was backed up most recently?",
    "Which snapshot is the largest?",
    "Which snapshots contain PDF files?",
  ],
  directory: [
    "What is in this directory?",
    "Which are the largest files here?",
    "What changed here since the previous snapshot?",
  ],
  file: [
    "What is this file about?",
    "How did this file change across snapshots?",
    "Which other snapshots hold this file?",
  ],
  errors: [
    "What went wrong in this snapshot?",
    "Which paths failed, and why?",
    "Did earlier snapshots have the same errors?",
  ],
  compare: [
    "What changed between these snapshots?",
    "Which files were added or removed?",
    "Summarize the changes by directory.",
  ],
} as const;

export function suggestionsFor(page: AssistantPage): readonly string[] {
  switch (page.name) {
    case "browse":
      return page.file ? SUGGESTIONS.file : SUGGESTIONS.directory;
    case "errors":
      return SUGGESTIONS.errors;
    case "compare":
      return SUGGESTIONS.compare;
    default:
      return SUGGESTIONS.repository;
  }
}
