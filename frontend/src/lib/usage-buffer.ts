// Serialize writes and acknowledge only the exact values included in each snapshot.
export const createUsageFlusher = <K, V>(
    read: () => Map<K, V>,
    acknowledge: (snapshot: Map<K, V>) => void,
    write: (snapshot: Map<K, V>) => Promise<unknown>,
) => {
    let inFlight: Promise<void> | null = null
    return (): Promise<void> => {
        if (inFlight) return inFlight
        const run = async () => {
            while (read().size > 0) {
                const snapshot = new Map(read())
                await write(snapshot)
                acknowledge(snapshot)
            }
        }
        inFlight = run().finally(() => { inFlight = null })
        return inFlight
    }
}
