package roles

var builtinRoles = []RoleDefinition{
	{
		Name:   RoleAdmin,
		Labels: Text{De: "Administrator", En: "Administrator"},
		Description: Text{
			De: "Verwaltet Benutzer und Einstellungen und sieht das Audit.",
			En: "Manages users and settings and can view the audit log.",
		},
	},
	{
		Name:   RoleArchivist,
		Labels: Text{De: "Archivar", En: "Archivist"},
		Description: Text{
			De: "Pflegt Ablagen, Archivierung und Aufbewahrung.",
			En: "Maintains stores, archiving and retention.",
		},
	},
	{
		Name:   RoleClerk,
		Labels: Text{De: "Sachbearbeiter", En: "Clerk"},
		Description: Text{
			De: "Erfasst und bearbeitet Dokumente im Tagesgeschäft.",
			En: "Creates and edits documents in day-to-day work.",
		},
	},
	{
		Name:   RoleReader,
		Labels: Text{De: "Leser", En: "Reader"},
		Description: Text{
			De: "Liest freigegebene Dokumente, ohne sie zu ändern.",
			En: "Reads released documents without changing them.",
		},
	},
}
