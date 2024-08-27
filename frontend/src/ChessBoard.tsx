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
                setIsLocked(true);
                stockFishMove(engine.fen()).then(() => {
                    setIsLocked(false); // Unlock the board after handling the API response
                }).catch(error => {
                    console.error('Error in API:', error);
                    setIsLocked(false); // Unlock the board in case of error
                });
                return true;
            }
        } catch (error) {
            console.error("Error making move:", error);
        }
        return false;
    };
    const stockFishMove = async (board: string): Promise<void> => {
        try {
            const response = await fetch("http://localhost:8081/engine-move", {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    depth: 12,
                    fen: board,
                }),
            });

            if (!response.ok) {
                throw new Error('Network response was not ok');
            }

            const data = await response.json();
            console.log('API response:', data);

            // Extract the 'bestmove' field
            const bestMove = data.bestmove;
            const success = data.success;

            if (bestMove && success) {
                engine.load(board)
                engine.move(bestMove)
                setBoardState(engine.fen())
            }
            else {
                throw new Error('API response was not ok');
            }

        } catch (error) {
            console.error('Error calling the API:', error);
        }
    };
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
