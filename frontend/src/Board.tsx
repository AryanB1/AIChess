import React, { useState } from 'react';
import { Chessboard } from 'react-chessboard';
import { Chess } from 'chess.js';

function PlayRandomMoveEngine() {
    const engine = new Chess();
    const [boardState, setBoardState] = useState(engine.fen());
    const playRandomMove = () => {
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
        setBoardState(engine.fen());
    };
    const onDrop = (sourceSquare: string, targetSquare: string): boolean => {
        engine.load(boardState)
        try {
            const move = engine.move({
                from: sourceSquare,
                to: targetSquare,
            });
            if (move) {
                setBoardState(engine.fen());
                return true;
            }
        } catch (error) {
            console.error("Error making move:", error);
        }
        return false;
    };

    return (
        <div style={{ width: '500px' }}>
            <Chessboard
                id="BasicBoard"
                boardOrientation="black"
                showBoardNotation={true}
                position={boardState}
                onPieceDrop={onDrop}
            />
            <button onClick={playRandomMove}>Play Random Move</button>
        </div>
    );
}

export default PlayRandomMoveEngine;
