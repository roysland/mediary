# Onboarding
We need to create several onboarding screens.
* Simple explanation of how good and secure passkeys are
* Language selection
* Adding trackables, predefined or custom
* Making audio recordings
* Navigation explanation, difference between just going to trackables, and adding trackable to an entry.

# Update/Alert system
After an update, user is met with a dismissable alert on the front page. Dismissing it will never show it again. This box is to inform about new or changed features.

# Service worker
Set up a service worker
* There should be a very high threshold for adding notifications. This is not the kind of app that should disturb the user. However, we do need to encourage keeping the diary. 
* Proper cache of javascript and css and images.

# Safe database schema upgrade
Some way to migrate contents if the database changes significantly. 

# Image upload
User should be able to upload an image. Use case: new rash developed, can take picture of it.

# Vector sqlite
Eventually, it would make more sense to create a vectorized search for entries, since the wording might differ for symptoms that are similar
- Which embedding to use?
- Which reasoning model?
- local models vs online services. (we have security to think of. And costs)

(Current VM is on Oracle Cloud, a VM.Standard.A1.Flex with 2 OCPU and 8GB ram. Can be upgraded without cost to 4ocpu and 24gb ram)

## Vectorized graphs?
Can we instead of just relying on the trackable values, create a visualization of vectors?
- Create a cluster map using an algorithm like UMAP or t-SNE. 

## Suggested symptoms
Based on vectors, can we make suggestions to what could be useful to track?
