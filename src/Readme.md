Approaches considered:
1. For each item, iterate over all its ingredients, and check if the desired quantity is available. 
    If all ingredients have desired quantity available, then update the resources accordingly.
    Issue - prone to race-conditions

2. To prevent race condition, use mutexes to serialize all operations - 
    checking if deisred quantity is available, then updating it.
    Issues - hampers concurrency. 

3. For each item, iterate over ingredients. If ingredient is available, consume it.
    Then do similarly for other ingredients. Now - for an item, we hit a case
    where a later ingredient doesn't have enough resource, then rollback already performed updates.
    
    Con: If another request arrives, it will see the wrong state of the system 
    [ since the other request could possibly rollback ], and it will report that there was insufficient resource
    
    Pro: We can use ingredient-level mutexes to achieve some degree of concurrency.
  

Approach used:
We use reservations to prevent a false state of the system [ explained in #3 ]
Whenever we recieve request to pour a drink, we try to take reservation for the item quantities if possible.

For example, 
consider "milk" ingredient has quantity = 10 currently
Now, a request for an item came - which needs 5 units of "milk"

Since 5 units of milk is available, we take a reservation for those 5 units.
Note: the milk quantity still stays as 10.

If all the reservations were possible, we then actually reflect these reservations in the resources 
[ by updating the resource quantity, and through use of mutexes, serializing the updates for each ingredient]

Then we delete all created reservations.

Now suppose another request would have come - demanding 5 units of milk.
We will check if ( currently-available(10) - reserved(5) >= demand ). 
If yes, this new request can also acquire reservation for 5 units of milk.

However, if the new request demands for 6 units, we see that this much quantity is not readily available.
But - there is a possibility that the request that has currently reserved 5 units of milk - 
could possibly fail after checking the next ingredients [ if enough quantity is unavailable]. 

So we return an error saying that - resource is temporarily busy/unavailable.
Then we retry such requests with appropriate backoffs.

Pro: 
1. No request sees a wrong state of the system [ reporting that an ingredient is unavailable, even though it is available after some time ]
2. We can use ingredient-level mutexes to achieve some degree of concurrency.
 