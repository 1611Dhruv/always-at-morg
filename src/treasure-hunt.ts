export interface Clue {
    id: number;
    riddle: string;
    answer: string; // In a real app, this might be hashed
    hint?: string;
}

export class TreasureHunt {
    private clues: Clue[];
    private currentStep: number = 0;
    private isCompleted: boolean = false;

    constructor(clues: Clue[]) {
        this.clues = clues;
    }

    public getCurrentClue(): string {
        if (this.isCompleted) {
            return "Congratulations! You have found the treasure!";
        }
        return this.clues[this.currentStep].riddle;
    }

    public submitAnswer(userAnswer: string): boolean {
        if (this.isCompleted) return false;

        const currentClue = this.clues[this.currentStep];
        
        // Simple case-insensitive comparison
        if (userAnswer.trim().toLowerCase() === currentClue.answer.toLowerCase()) {
            this.advance();
            return true;
        }
        return false;
    }

    public getHint(): string {
        if (this.isCompleted) return "You've already won.";
        return this.clues[this.currentStep].hint || "No hint available.";
    }

    private advance() {
        this.currentStep++;
        if (this.currentStep >= this.clues.length) {
            this.isCompleted = true;
        }
    }

    public getProgress(): number {
        return this.currentStep;
    }
}
