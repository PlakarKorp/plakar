import { test, expect } from "@playwright/test";

const DASHBOARD_URL_ENV = "PLAKAR_UI_URL";
const DASHBOARD_URL = process.env[DASHBOARD_URL_ENV];
const DASHBOARD_HEADING = "Your dashboard";
const REMOVED_METRIC = "Logical size";
const REPLACEMENT_METRIC = "Storage size";

test("dashboard shows storage size instead of logical size", async ({ page }) => {
  if (!DASHBOARD_URL) {
    throw new Error(`${DASHBOARD_URL_ENV} is required`);
  }

  await page.goto(DASHBOARD_URL, { waitUntil: "networkidle" });
  await expect(page.getByText(DASHBOARD_HEADING)).toBeVisible();
  await expect(page.getByText(REPLACEMENT_METRIC)).toBeVisible();
  await expect(page.getByText(REMOVED_METRIC)).toHaveCount(0);

  console.log(
    `Browser verification: tested in Chrome via Playwright; route=/; checked ${REPLACEMENT_METRIC} is visible and ${REMOVED_METRIC} is absent.`,
  );
});
