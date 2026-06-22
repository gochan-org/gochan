import { describe, it, expect, beforeEach, jest, beforeAll } from "@jest/globals"
(global as any).jQuery = require("jquery"); // Jest doesn't detect jQuery
// import { initStaff } from "../ts/management/manage";
import { MockResponse } from "./util";

const mockActions:StaffAction[] = [
	{id: "mock3", title: "Admin-only actions", perms: 3, jsonOutput: 1},
	{id: "mock2", title: "Mod+Admin actions", perms: 2, jsonOutput: 1},
	{id: "mock1", title: "JanitorMod+Admin actions", perms: 1, jsonOutput: 1},
	{id: "staffinfo", title: "", perms: 0, jsonOutput: 2},
];

const baseStaff:StaffInfo[] = [
	{username: "admin", rank: 3, actions: mockActions},
	{username: "moderator", rank: 2, actions: mockActions.filter(a => a.perms <= 2)},
	{username: "janitor", rank: 1, actions: mockActions.filter(a => a.perms <= 1)},
	{username: "", rank: 0}, // not logged in
];


describe("Staff info", function() {
	let consoleSpy: jest.SpiedFunction<typeof console.error>;
	beforeAll(() => {
		consoleSpy = jest.spyOn(console, "error");
	});
	beforeEach(() => {
		jest.clearAllMocks();
		jest.resetModules();
		global.fetch = jest.fn<any>();
	});

	baseStaff.map(staff => {
		it(`gets staff info for ${staff.username == ""?"logged out user":staff.username}`, async () => {
			const mockResponse = new MockResponse<StaffInfo>("/manage/staffinfo", JSON.stringify(staff), "application/json");
			(global.fetch as jest.Mock<any>).mockResolvedValue(mockResponse);
			const { initStaff } = await import("../ts/management/manage");
			const result = await initStaff();
			expect(result).toEqual(staff);

			const resultCached = await initStaff();
			expect(resultCached).toEqual(staff);
			expect(global.fetch).toHaveBeenCalledTimes(1);
		});
	});

	it("gets staff info fails", async function() {
		consoleSpy = jest.spyOn(console, "error");
		const { initStaff } = await import("../ts/management/manage");
		const mockResponse = new MockResponse<StaffInfo>("/manage/staffinfo", "Internal Server Error", "text/plain", false, 500, "Internal Server Error");
		mockResponse.ok = false;
		(global.fetch as jest.Mock<any>).mockRejectedValue(mockResponse);
		expect(initStaff()).rejects.toThrow("Error getting staff info: Internal Server Error");
	});
});