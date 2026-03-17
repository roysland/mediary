# 1. New entry on front page
Remove entry list on front page. It adds noise to the UX. 
The submit button should have a working "data-submit-state-button". 
After submitting, the form should only be cleared as it is now. Only a visual cue that the entry is saved (using the data-submit-state-button)

# 2. Entries list
The context menu buttons do nothing. 
Add trackable button should display a new dialog with the class sheet (fullscreen), and show the same screen as /trackables. 
Delete button should use a native javascript confirm alert. If confirmed, then delete entry. 

# 3. Trackables
By removing the inline oninput listener, the update to the output is broken. If it's hard to fix, reimplement the inline function. 

# 4. Add trackable
Writing inside the name input does not filter the list.
Choosing a preset does not populate the inputs. 
The "sensitive" checkbox isn't connected to the visibility of the form element. It should only be displayed if the checkbox is checked. 

# 5. Settings
Choosing a theme is currently only a database value. It doesn't do anything in the UI. Implement this.
The "Clear all data" button successfully opens the popover. The popover has another button with the same name. This button inside popover should call a native javascript confirm with a warning message. If confirmed, all data related to the current user should be deleted. 

# 6. Bottom nav
Make the page visible. Add proper aria attributes, and add a the css variable --color-text to the active element.