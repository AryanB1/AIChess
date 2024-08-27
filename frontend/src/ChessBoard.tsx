import React, { useState } from 'react';
import { Chessboard } from 'react-chessboard';
import { Chess } from 'chess.js';

function ChessBoard() {
    const engine = new Chess();
    type BoardOrientation = 'white' | 'black';
    const [boardState, setBoardState] = useState(engine.fen());
    const [boardOrientationState, setBoardOrientationState] = useState<BoardOrientation>('white');
    const [isLocked, setIsLocked] = useState(false);
    const [gameState, setGameState] = useState(false);
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
        if(engine.isGameOver()) {
            setIsLocked(true)
            setGameState(true)
            return true;
        }
        console.log(legalMoves);
        console.log(engine.fen())
    };
    const onDrop = (sourceSquare: string, targetSquare: string): boolean => {
        console.log(engine.moves({verbose: true}))
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
                if(engine.isGameOver()) {
                    setIsLocked(true)
                    setGameState(true)
                    return true;
                }
                stockFishMove(engine.fen()).then(() => {
                    setIsLocked(false); // Unlock the board after handling the API response
                }).catch(error => {
                    console.error('Error in API:', error);
                    setIsLocked(false); // Unlock the board in case of error
                });
                if(engine.isGameOver()){
                    console.log("game won!")
                    setGameState(true)
                    setIsLocked(true)
                }
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
                if(engine.isGameOver()) {
                    setIsLocked(true)
                    setGameState(true)
                }
            }
            else {
                throw new Error('API response was not ok');
            }

        } catch (error) {
            console.error('Error calling the API:', error);
        }
    };
    const ResetGame = () => {
        engine.reset()
        setBoardState(engine.fen())
        setIsLocked(false)
        setGameState(false)
    }
    return (
        <div style={{ backgroundColor: '#f4f4f9', minHeight: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '20px' }}>
            <div style={{ marginBottom: '20px' }}>
                {gameState && <div style={{ fontSize: '18px', fontWeight: 'bold', color: 'red' }}>Game is over!</div>}
                {!gameState && <div style={{ fontSize: '18px', fontWeight: 'bold', color: 'green' }}>Game in progress</div>}
            </div>
            <div style={{ width: '500px', marginBottom: '20px' }}>
                <Chessboard
                    id="BasicBoard"
                    boardOrientation={boardOrientationState}
                    showBoardNotation={true}
                    position={boardState}
                    onPieceDrop={onDrop}
                />
            </div>
            <div style={{ display: 'flex', gap: '10px' }}>
                <button
                    onClick={playRandomMove}
                    disabled={isLocked}
                    style={{ padding: '10px 20px', backgroundColor: '#007bff', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}
                >
                    Play Random Move
                </button>
                <button
                    onClick={flipBoard}
                    style={{ padding: '10px 20px', backgroundColor: '#28a745', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}
                >
                    Flip Board
                </button>
                <button
                    onClick={ResetGame}
                    style={{ padding: '10px 20px', backgroundColor: '#dc3545', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}
                >
                    Reset Game
                </button>
            </div>
        </div>
    );
}

export default ChessBoard;
