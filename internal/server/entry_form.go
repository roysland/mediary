package server

type entryFormViewData struct {
	FormAction         string
	EntryID            int64
	UseHTMX            bool
	UseDialogRefresh   bool
	EntryDate          string
	ShowEntryDate      bool
	ShowImageUpload    bool
	ShowPastDayWarning bool
	NoteText           string
	IsPrivate          bool
}

func buildEntryFormViewData(formAction, entryDate, today string, showEntryDate, useHTMX, useDialogRefresh bool) entryFormViewData {
	return entryFormViewData{
		FormAction:         formAction,
		UseHTMX:            useHTMX,
		UseDialogRefresh:   useDialogRefresh,
		EntryDate:          entryDate,
		ShowEntryDate:      showEntryDate,
		ShowImageUpload:    true,
		ShowPastDayWarning: entryDate != "" && today != "" && entryDate < today,
	}
}
