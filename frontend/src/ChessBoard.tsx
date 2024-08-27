import React, { useState } from 'react';
import { Chessboard } from 'react-chessboard';
import { Chess } from 'chess.js';

function ChessBoard() {
    const engine = new Chess();
    type BoardOrientation = 'white' | 'black';
    const [boardState, setBoardState] = useState(engine.fen());
    const [boardOrientationState, setBoardOrientationState] = useState<BoardOrientation>('white');
    const [isLocked, setIsLocked] = useState(false);
    const flipBoard = () => {
        setBoardOrientationState((prev) => (prev === 'white' ? 'black' : 'white'));
    };
    const playRandomMove = () => {
        if (isLocked) return;
        engine.load(boardState)
        const legalMoves = engine.moves({verbose:true});
        let randomIndex = Math.floor(Math.random() * legalMoves.length);
        while (true) {
            let validator = onDrop(legalMoves[randomIndex].from, legalMoves[randomIndex].to);
            if (validator) break;
            randomIndex = Math.floor(Math.random() * legalMoves.length);
        }
        console.log(legalMoves);
        console.log(engine.fen())
    };
    const onDrop = (sourceSquare: string, targetSquare: string): boolean => {
        if (isLocked) return false;
        engine.load(boardState)
        try {
            const move = engine.move({
                from: sourceSquare,
                to: targetSquare,
            });
            if (move) {
                setBoardState(engine.fen());
                setIsLocked(true)
                stockFishMove()
                return true;
            }
        } catch (error) {
            console.error("Error making move:", error);
        }
        return false;
    };
    const stockFishMove = (): string => {

        return ""
    }
    return (
        <div style={{ width: '500px', paddingLeft: '300px', paddingTop: '100px'}}>
            <Chessboard
                id="BasicBoard"
                boardOrientation={boardOrientationState}
                showBoardNotation={true}
                position={boardState}
                onPieceDrop={onDrop}
            />
            <button onClick={playRandomMove}>Play Random Move</button>
            <button onClick={flipBoard}>Flip Board</button>
        </div>
    );
}

export default ChessBoard;
